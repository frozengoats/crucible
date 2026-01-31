package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const ArtifactType = "application/vnd.crucible.recipe.v1"

type RemoteDescriptor struct {
	ArtifactType string `json:"artifactType"`
}

type Credentials struct {
	Username string `json:"username"`
	Password []byte `json:"password"`
}

func IsOciUrl(url string) bool {
	return strings.HasPrefix(url, "oci://")
}

func GetOciStoragePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "crucible", "recipes"), nil
}

type ImageDescriptor struct {
	Registry   string
	Repository string
	Tag        string
}

func (id *ImageDescriptor) Url() string {
	return fmt.Sprintf("oci://%s", id.Ref())
}

func (id *ImageDescriptor) Ref() string {
	return fmt.Sprintf("%s/%s:%s", id.Registry, id.Repository, id.Tag)
}

func (id *ImageDescriptor) StoragePath() (string, error) {
	ociDownloadDir, err := GetOciStoragePath()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s/%s", ociDownloadDir, id.Registry, id.Repository, id.Tag), nil
}

func (id *ImageDescriptor) GetDigest() (string, error) {
	storagePath, err := id.StoragePath()
	if err != nil {
		return "", err
	}

	dBytes, err := os.ReadFile(filepath.Join(storagePath, "digest.sha"))
	if err != nil {
		// error means no file, doesn't exist
		return "", nil
	}

	return strings.Trim(string(dBytes), "\n\t "), nil
}

func (id *ImageDescriptor) UpdateDigest(digest string) error {
	storagePath, err := id.StoragePath()
	if err != nil {
		return err
	}
	digestPath := filepath.Join(storagePath, "digest.sha")
	return os.WriteFile(digestPath, []byte(digest), 0664)
}

func NewImageDescriptor(url string) (*ImageDescriptor, error) {
	if !strings.HasPrefix(url, "oci://") {
		return nil, fmt.Errorf("url must begin with oci://")
	}

	strippedUrl := strings.TrimPrefix(url, "oci://")
	urlParts := strings.Split(strippedUrl, ":")
	if len(urlParts) > 2 {
		return nil, fmt.Errorf("url %s is malformed", url)
	}
	var tag string
	if len(urlParts) == 1 {
		tag = "latest"
	} else {
		tag = urlParts[1]
	}

	if tag == "" {
		tag = "latest"
	}

	parts := strings.SplitN(urlParts[0], "/", 2)
	return &ImageDescriptor{
		Registry:   parts[0],
		Repository: parts[1],
		Tag:        tag,
	}, nil
}

func LoadCredentials() (map[string]*Credentials, error) {
	credentials := map[string]*Credentials{}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	credPath := filepath.Join(homeDir, ".config", "crucible")
	err = os.MkdirAll(credPath, 0700)
	if err != nil {
		return nil, err
	}

	credFilePath := filepath.Join(credPath, "credentials.json")
	_, err = os.Stat(credFilePath)
	if err != nil {
		return credentials, nil
	}

	credBytes, err := os.ReadFile(credFilePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(credBytes, &credentials)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

func SaveCredentials(creds map[string]*Credentials) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	credPath := filepath.Join(homeDir, ".config", "crucible")
	err = os.MkdirAll(credPath, 0700)
	if err != nil {
		return err
	}

	cBytes, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(credPath, "credentials.json"), cBytes, 0600)
}

// getAuthCredentials returns nil when credentials haven't been defined
func getAuthCredentials(domain string) (*auth.Credential, error) {
	ociUsername := os.Getenv("OCI_REGISTRY_USERNAME")
	ociPassword := os.Getenv("OCI_REGISTRY_PASSWORD")
	if ociUsername != "" && ociPassword != "" {
		return &auth.Credential{
			Username: ociUsername,
			Password: ociPassword,
		}, nil
	}

	credMap, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	if credMap == nil {
		return nil, nil
	}

	c, ok := credMap[domain]
	if !ok {
		return nil, nil
	}

	return &auth.Credential{
		Username: c.Username,
		Password: string(c.Password),
	}, nil
}

func applyDomainCreds(imageDescriptor *ImageDescriptor, repo *remote.Repository) error {
	creds, err := getAuthCredentials(imageDescriptor.Registry)
	if err != nil {
		return err
	}

	if creds != nil {
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.NewCache(),
			Credential: func(ctx context.Context, hostport string) (auth.Credential, error) {
				return *creds, nil
			},
		}
	}

	return nil
}

func Publish(recipePath string, registryPrefix string, name string, version string) (*ImageDescriptor, error) {
	var fqName string
	if !strings.HasPrefix(name, "recipe.") {
		fqName = fmt.Sprintf("recipe.%s", name)
	} else {
		fqName = name
	}

	imageDescriptor, err := NewImageDescriptor(fmt.Sprintf("oci://%s/%s:%s", registryPrefix, fqName, version))
	if err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	ociTempDir := filepath.Join(homeDir, "crucible", "tmp")

	fs, err := file.New(ociTempDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = fs.Close()
	}()

	var fds []v1.Descriptor
	fd, err := fs.Add(context.Background(), name, "", recipePath)
	if err != nil {
		return nil, err
	}
	fds = append(fds, fd)

	opts := oras.PackManifestOptions{
		Layers: fds,
	}
	manifestDesc, err := oras.PackManifest(context.Background(), fs, oras.PackManifestVersion1_1, ArtifactType, opts)
	if err != nil {
		return nil, err
	}

	if err = fs.Tag(context.Background(), manifestDesc, version); err != nil {
		return nil, err
	}

	repo, err := remote.NewRepository(imageDescriptor.Ref())
	if err != nil {
		return nil, err
	}

	err = applyDomainCreds(imageDescriptor, repo)
	if err != nil {
		return nil, err
	}

	if _, err = oras.Copy(context.Background(), fs, version, repo, version, oras.DefaultCopyOptions); err != nil {
		return nil, err
	}
	return imageDescriptor, nil
}

// Download downloads the artifact
func Download(imageDescriptor *ImageDescriptor, force bool) error {
	ociDownloadDir, err := imageDescriptor.StoragePath()
	if err != nil {
		return err
	}

	fs, err := file.New(ociDownloadDir)
	if err != nil {
		return err
	}

	repo, err := remote.NewRepository(imageDescriptor.Ref())
	if err != nil {
		return err
	}

	if err = applyDomainCreds(imageDescriptor, repo); err != nil {
		return err
	}

	localDigest, err := imageDescriptor.GetDigest()
	if err != nil {
		return err
	}

	remoteDescOras, fetchBytes, err := oras.FetchBytes(context.Background(), repo, imageDescriptor.Tag, oras.DefaultFetchBytesOptions)
	if err != nil {
		return err
	}

	// this is needed b/c at the time, ORAS does not unmarshal the artifact type to the structure
	remoteDesc := &RemoteDescriptor{}
	if err = json.Unmarshal(fetchBytes, remoteDesc); err != nil {
		return err
	}

	if localDigest != "" && !force {
		// if the new one matches what we have on disk, return
		if remoteDescOras.Digest.String() == localDigest {
			fmt.Printf("skipping download because a matching local copy already exists\n")
			return nil
		}
	}

	if remoteDesc.ArtifactType != ArtifactType {
		return fmt.Errorf("artifact with type %s was not a crucible recipe", remoteDesc.ArtifactType)
	}

	if err = os.RemoveAll(ociDownloadDir); err != nil {
		return err
	}
	manifestDescriptor, err := oras.Copy(context.Background(), repo, imageDescriptor.Tag, fs, imageDescriptor.Tag, oras.DefaultCopyOptions)
	if err != nil {
		return err
	}

	if err = imageDescriptor.UpdateDigest(manifestDescriptor.Digest.String()); err != nil {
		return err
	}

	fmt.Printf("downloaded recipe %s\n", imageDescriptor.Repository)
	return nil
}

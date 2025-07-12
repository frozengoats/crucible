package kvstore

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var indexIdentifier = regexp.MustCompile(`([^[]+)\[(-?\d+)\]`)

func parseArrayKey(test string) (string, int, bool) {
	match := indexIdentifier.FindAllStringSubmatch(test, -1)
	if len(match) == 0 {
		return "", 0, false
	}

	index, err := strconv.ParseInt(match[0][2], 10, 64)
	if err != nil {
		return "", 0, false
	}

	return match[0][1], int(index), true
}

func reCastStringArray(value []string) []any {
	newArr := make([]any, len(value))
	for i, v := range value {
		newArr[i] = v
	}
	return newArr
}

func reCastIntArray(value []int) []any {
	newArr := make([]any, len(value))
	for i, v := range value {
		newArr[i] = v
	}
	return newArr
}

func reCastFloatArray(value []float64) []any {
	newArr := make([]any, len(value))
	for i, v := range value {
		newArr[i] = v
	}
	return newArr
}

func verifySlice(data []any) error {
	for _, v := range data {
		switch t := v.(type) {
		case bool:
		case int:
		case float64:
		case string:
		case nil:
		case []any:
			err := verifySlice(t)
			if err != nil {
				return err
			}
		case map[string]any:
			err := verifyMapping(t)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected value type %v", v)
		}
	}

	return nil
}

func verifyMapping(data map[string]any) error {
	for k, v := range data {
		switch t := v.(type) {
		case bool:
		case int:
		case float64:
		case string:
		case nil:
		case []any:
			err := verifySlice(t)
			if err != nil {
				return err
			}
		case map[string]any:
			err := verifyMapping(t)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected value type %v at key %s", v, k)
		}
	}

	return nil
}

func deepCopySlice(src []any) []any {
	newSlice := make([]any, len(src))
	for i, v := range src {
		switch t := v.(type) {
		case []any:
			newSlice[i] = deepCopySlice(t)
		case map[string]any:
			newSlice[i] = deepCopyMap(t)
		default:
			// copy by value
			newSlice[i] = v
		}
	}

	return newSlice
}

func deepCopyMap(src map[string]any) map[string]any {
	newMap := map[string]any{}
	for k, v := range src {
		switch t := v.(type) {
		case []any:
			newMap[k] = deepCopySlice(t)
		case map[string]any:
			newMap[k] = deepCopyMap(t)
		default:
			// copy by value
			newMap[k] = v
		}
	}

	return newMap
}

func copyOver(target map[string]any, src map[string]any) {
	for k, v := range src {
		t, ok := target[k]
		if !ok {
			target[k] = v
			continue
		}

		// key exists on the src side
		switch st := v.(type) {
		case map[string]any:
			// if the source value is a map, check if the target side is also a map
			switch tt := t.(type) {
			case map[string]any:
				// since both the target and source values are maps, recurse
				copyOver(tt, st)
				continue
			}
		}

		target[k] = v
	}
}

type Store struct {
	data map[string]any
}

// NewStore returns an empty but initialized store object
func NewStore() *Store {
	return &Store{
		data: map[string]any{},
	}
}

// FromMapping returns a new store object from a standard mapping
func FromMapping(mapping map[string]any) (*Store, error) {
	if mapping == nil {
		mapping = map[string]any{}
	}
	err := verifyMapping(mapping)
	if err != nil {
		return nil, err
	}

	return &Store{
		data: mapping,
	}, nil
}

// get traverses the store and attempts to retrieve value indicated by the hierarchical
// namespace.
func (s *Store) get(namespace ...any) (any, bool) {
	var root any

	root = s.data
	total_ns_keys := len(namespace)
	for i, ns := range namespace {
		switch nsTyped := ns.(type) {
		case string:
			rootMap, ok := root.(map[string]any)
			if !ok {
				return nil, false
			}
			root, ok = rootMap[nsTyped]
			if !ok {
				return nil, false
			}
		case int:
			rootSlice, ok := root.([]any)
			if !ok {
				return nil, false
			}
			if nsTyped >= len(rootSlice) {
				return nil, false
			}

			if nsTyped < 0-len(rootSlice) {
				return nil, false
			}
			if nsTyped < 0 {
				nsTyped = len(rootSlice) + nsTyped
			}
			root = rootSlice[nsTyped]
		default:
			return nil, false
		}

		// if this is the final namespace key, return the value
		if i == total_ns_keys-1 {
			return root, true
		}
	}

	// this is unreachable code
	return nil, false
}

// Get attempts to return the value stored at namespace, where namespace is a
// hierarchical ordered list of keys nested from left at the root, to right.
// If the value does NOT exist, a nil interface will be returned.
func (s *Store) Get(namespace ...any) any {
	v, _ := s.get(namespace...)
	return v
}

func (s *Store) Exists(namespace ...any) bool {
	_, ok := s.get(namespace...)
	return ok
}

// Set sets a value in the store at the hierarchical position indicated by namespace,
// creating or modifying (destructively) the hierarchy in order to support the operation.
func (s *Store) Set(value any, namespace ...any) error {
	var root any
	if len(namespace) < 1 {
		return fmt.Errorf("must provide at least 1 level of namespacing")
	}
	switch t := value.(type) {
	case map[string]any:
		err := verifyMapping(t)
		if err != nil {
			return err
		}
	case []any:
		err := verifySlice(t)
		if err != nil {
			return err
		}
	case []int:
		// re-cast to []any
		v := reCastIntArray(t)
		err := verifySlice(v)
		if err != nil {
			return err
		}
		value = v
	case []float64:
		// re-cast to []any
		v := reCastFloatArray(t)
		err := verifySlice(v)
		if err != nil {
			return err
		}
		value = v
	case []string:
		// re-cast to []any
		v := reCastStringArray(t)
		err := verifySlice(v)
		if err != nil {
			return err
		}
		value = v
	}

	root = s.data
	total_ns_keys := len(namespace)
	for i, ns := range namespace {
		if i == total_ns_keys-1 {
			switch nsTyped := ns.(type) {
			case string:
				rootMap, ok := root.(map[string]any)
				if !ok {
					return fmt.Errorf("attempted to set a key on a non-mapping type location")
				}
				rootMap[nsTyped] = value
			case int:
				rootSlice, ok := root.([]any)
				if !ok {
					return fmt.Errorf("attempted to set a value by index on a non-array type location")
				}
				if nsTyped < 0 || nsTyped >= len(rootSlice) {
					return fmt.Errorf("index error on final destination value")
				}
				rootSlice[nsTyped] = value
			}
			return nil
		}

		switch nsTyped := ns.(type) {
		case string:
			rootMap, ok := root.(map[string]any)
			if !ok {
				return fmt.Errorf("attempted to set a key on a non-mapping type location")
			}
			next, ok := rootMap[nsTyped]
			if ok {
				root = next
				continue
			}

			newNode := map[string]any{}
			rootMap[nsTyped] = newNode
			root = newNode
		case int:
			rootSlice, ok := root.([]any)
			if !ok {
				return fmt.Errorf("attempted to set a value by index on a non-array type location")
			}
			if nsTyped < 0 || nsTyped >= len(rootSlice) {
				return fmt.Errorf("index error on array")
			}
			root = rootSlice[nsTyped]
		}
	}

	return nil
}

// GetMapping retrieves a mapping value from the store at namespace or nil
func (s *Store) GetMapping(namespace ...any) map[string]any {
	v, ok := s.get(namespace...)
	if !ok {
		return nil
	}
	mp, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return mp
}

// GetStore retrieves a mapping value from the store and returns a new store object.  A copy
// is not made, and no additonal validation is performed, this is a fast operation.
func (s *Store) GetStore(namespace ...any) *Store {
	v, ok := s.get(namespace...)
	if !ok {
		return nil
	}
	mp, ok := v.(map[string]any)
	if !ok {
		return NewStore()
	}
	return &Store{
		data: mp,
	}
}

// GetInt retrieves an integer value from the store at namespace or the zero value
// if it cannot be located
func (s *Store) GetInt(namespace ...any) int {
	var value int
	v, ok := s.get(namespace...)
	if !ok {
		return value
	}

	t, ok := v.(int)
	if ok {
		return t
	}

	return value
}

// GetFloat retrieves a float value from the store at namespace or the zero value
// if it cannot be located
func (s *Store) GetFloat(namespace ...any) float64 {
	var value float64
	v, ok := s.get(namespace...)
	if !ok {
		return value
	}

	t, ok := v.(float64)
	if ok {
		return t
	}

	return value
}

// GetString retrieves a string value from the store at namespace or the zero value
// if it cannot be located
func (s *Store) GetString(namespace ...any) string {
	var value string
	v, ok := s.get(namespace...)
	if !ok {
		return value
	}

	t, ok := v.(string)
	if ok {
		return t
	}

	return value
}

// GetBool retrieves a boolean value from the store at namespace or the zero value
// if it cannot be located
func (s *Store) GetBool(namespace ...any) bool {
	var value bool
	v, ok := s.get(namespace...)
	if !ok {
		return value
	}

	t, ok := v.(bool)
	if ok {
		return t
	}

	return value
}

// GetArray retrieves an array value from the store at namespace or the zero value
// if it cannot be located.  The zero value for an array is nil
func (s *Store) GetArray(namespace ...any) []any {
	var value []any
	v, ok := s.get(namespace...)
	if !ok {
		return value
	}

	t, ok := v.([]any)
	if ok {
		return t
	}

	return value
}

// GetMappingArray retrieves a mapping array value from the store at namespace or the zero value
// if it cannot be located.  The zero value for an array is nil
func (s *Store) GetMappingArray(namespace ...any) []map[string]any {
	var value []map[string]any
	v, ok := s.get(namespace...)
	if !ok {
		return value
	}

	t, ok := v.([]map[string]any)
	if ok {
		return t
	}

	return value
}

// GetStringArray retrieves and re-casts a string array value from the store at namespace or the zero value
// if it cannot be located.  The zero value for an array is nil
func (s *Store) GetStringArray(namespace ...any) []string {
	var empty []string
	var value []string
	v, ok := s.get(namespace...)
	if !ok {
		return empty
	}

	t, ok := v.([]any)
	if !ok {
		return empty
	}

	value = make([]string, len(t))
	for i, v := range t {
		finalV, ok := v.(string)
		if !ok {
			return empty
		}
		value[i] = finalV
	}
	return value
}

// GetIntArray retrieves and re-casts an integer array value from the store at namespace or the zero value
// if it cannot be located.  The zero value for an array is nil
func (s *Store) GetIntArray(namespace ...any) []int {
	var empty []int
	var value []int
	v, ok := s.get(namespace...)
	if !ok {
		return empty
	}

	t, ok := v.([]any)
	if !ok {
		return empty
	}

	value = make([]int, len(t))
	for i, v := range t {
		finalV, ok := v.(int)
		if !ok {
			return empty
		}
		value[i] = finalV
	}
	return value
}

// GetFloatArray retrieves and re-casts a float64 array value from the store at namespace or the zero value
// if it cannot be located.  The zero value for an array is nil
func (s *Store) GetFloatArray(namespace ...any) []float64 {
	var empty []float64
	var value []float64
	v, ok := s.get(namespace...)
	if !ok {
		return empty
	}

	t, ok := v.([]any)
	if !ok {
		return empty
	}

	value = make([]float64, len(t))
	for i, v := range t {
		finalV, ok := v.(float64)
		if !ok {
			return empty
		}
		value[i] = finalV
	}
	return value
}

// GetStoreArray is a convenient way of returning an array of mappings as an array of Store objects.
// Returning store objects allows for further processing without the overhead of the initial validation
// when constructing a new object from an external data source.  If the original data is not an array of
// mappings then an empty array will be returned.
func (s *Store) GetStoreArray(namespace ...any) []*Store {
	var empty []*Store
	var value []*Store
	v, ok := s.get(namespace...)
	if !ok {
		return empty
	}

	t, ok := v.([]any)
	if !ok {
		return empty
	}

	value = make([]*Store, len(t))
	for i, v := range t {
		finalV, ok := v.(map[string]any)
		if !ok {
			return empty
		}
		value[i] = &Store{
			data: finalV,
		}
	}
	return value
}

// Overlay returns a new store object which overlays the provided store onto this store
// This method isn't designed for high efficiency, but rather high code maintainability - for
// this reason the entire base is copied before determining which keys are actually needed in the final overlay
func (s *Store) Overlay(ovl *Store) *Store {
	newMap := deepCopyMap(s.data)
	overlayMap := deepCopyMap(ovl.data)
	copyOver(newMap, overlayMap)
	// since this is all vetted data, an error is impossible
	newStore, _ := FromMapping(newMap)
	return newStore
}

func (s *Store) DeepCopy() *Store {
	newMap := deepCopyMap(s.data)
	// since this is all vetted data, an error is impossible
	newStore, _ := FromMapping(newMap)
	return newStore
}

// ParseKey returns a namespace array from a namespace string
func ParseNamespaceString(key string) []any {
	var keys []any
	keyParts := strings.Split(key, ".")
	for _, kp := range keyParts {
		key, index, ok := parseArrayKey(kp)
		if !ok {
			keys = append(keys, kp)
		} else {
			keys = append(keys, key)
			keys = append(keys, index)
		}
	}

	return keys
}

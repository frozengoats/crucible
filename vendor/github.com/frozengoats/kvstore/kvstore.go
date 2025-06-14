package kvstore

import "fmt"

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
func (s *Store) get(namespace ...string) (any, bool) {
	root := s.data
	total_ns_keys := len(namespace)
	for i, ns := range namespace {
		next, ok := root[ns]
		if !ok {
			return nil, false
		}

		// if this is the final namespace key, return the value
		if i == total_ns_keys-1 {
			return next, true
		}

		switch t := next.(type) {
		case map[string]any:
			root = t
		default:
			return nil, false
		}
	}

	// this is unreachable code
	return nil, false
}

// Get attempts to return the value stored at namespace, where namespace is a
// hierarchical ordered list of keys nested from left at the root, to right.
// If the value does NOT exist, a nil interface will be returned.
func (s *Store) Get(namespace ...string) any {
	v, _ := s.get(namespace...)
	return v
}

func (s *Store) Exists(namespace ...string) bool {
	_, ok := s.get(namespace...)
	return ok
}

// Set sets a value in the store at the hierarchical position indicated by namespace,
// creating or modifying (destructively) the hierarchy in order to support the operation.
func (s *Store) Set(value any, namespace ...string) error {
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

	root := s.data
	total_ns_keys := len(namespace)
	for i, ns := range namespace {
		if i == total_ns_keys-1 {
			root[ns] = value
			return nil
		}

		next, ok := root[ns]
		if ok {
			switch t := next.(type) {
			case map[string]any:
				root = t
				continue
			}
		}

		// if type changed or a node was unavailable, add a new node
		newNode := map[string]any{}
		root[ns] = newNode
		root = newNode
	}

	return nil
}

// GetMapping retrieves a mapping value from the store at namespace or nil
func (s *Store) GetMapping(namespace ...string) map[string]any {
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

// GetInt retrieves an integer value from the store at namespace or the zero value
// if it cannot be located
func (s *Store) GetInt(namespace ...string) int {
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
func (s *Store) GetFloat(namespace ...string) float64 {
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
func (s *Store) GetString(namespace ...string) string {
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
func (s *Store) GetBool(namespace ...string) bool {
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
func (s *Store) GetArray(namespace ...string) []any {
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
func (s *Store) GetMappingArray(namespace ...string) []map[string]any {
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
func (s *Store) GetStringArray(namespace ...string) []string {
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
func (s *Store) GetIntArray(namespace ...string) []int {
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
func (s *Store) GetFloatArray(namespace ...string) []float64 {
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

// Overlay returns a new store object which overlays the provided store onto this store
// This method isn't designed for high efficiency, but rather high code maintainability - for
// this reason the entire base is copied before determining which keys are actually needed in the final overlay
func (s *Store) Overylay(ovl *Store) (*Store, error) {
	newMap := deepCopyMap(s.data)
	overlayMap := deepCopyMap(ovl.data)
	copyOver(newMap, overlayMap)
	return FromMapping(newMap)
}

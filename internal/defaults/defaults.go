package defaults

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func setInt(fv reflect.Value, value string) error {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	fv.SetInt(v)
	return nil
}

func setFloat(fv reflect.Value, value string) error {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	fv.SetFloat(v)
	return nil
}

func setBool(fv reflect.Value, value string) error {
	var bv bool
	lv := strings.ToLower(value)
	switch lv {
	case "t", "true", "1":
		bv = true
	case "f", "false", "0":
		bv = false
	default:
		return fmt.Errorf("string was not clearly representative of a boolean value")
	}

	fv.SetBool(bv)
	return nil
}

func applyValue(fv reflect.Value, value string) error {
	var err error
	switch fv.Kind() {
	case reflect.Int:
		err = setInt(fv, value)
	case reflect.Int64:
		err = setInt(fv, value)
	case reflect.Int32:
		err = setInt(fv, value)
	case reflect.Int16:
		err = setInt(fv, value)
	case reflect.Int8:
		err = setInt(fv, value)
	case reflect.Float64:
		err = setFloat(fv, value)
	case reflect.Float32:
		err = setFloat(fv, value)
	case reflect.String:
		fv.SetString(value)
	case reflect.Bool:
		err = setBool(fv, value)

	default:
		return fmt.Errorf("unexpected field type %s", fv.Kind().String())
	}

	return err
}

func ApplyDefaults(target any) error {
	t := reflect.TypeOf(target)
	v := reflect.ValueOf(target)
	k := t.Kind()

	var typ reflect.Type
	var ve reflect.Value
	if k == reflect.Pointer {
		typ = t.Elem()
		ve = v.Elem()
		//return fmt.Errorf("target must be a pointer type")
	} else {
		typ = reflect.TypeOf(target)
		ve = v
	}

	k = typ.Kind()
	if k != reflect.Struct {
		return fmt.Errorf("target must point to a struct type")
	}

	for i := range typ.NumField() {
		f := ve.Field(i)
		v := typ.Field(i)

		if v.Name[:1] != strings.ToUpper(v.Name[:1]) {
			// this is a private member, don't look at it
			continue
		}

		switch f.Kind() {
		case reflect.Pointer:
			if f.IsNil() {
				continue
			}
			ptr := f.Interface()
			val := reflect.ValueOf(ptr)
			elem := val.Elem()
			if elem.Kind() == reflect.Struct {
				err := ApplyDefaults(ptr)
				if err != nil {
					return err
				}

				continue
			}
		case reflect.Struct:
			// go deeper
			ifx := f.Addr().Interface()
			err := ApplyDefaults(ifx)
			if err != nil {
				return err
			}

			continue
		case reflect.Map:
			// go deeper
			for _, mapKey := range f.MapKeys() {
				mapValue := f.MapIndex(mapKey)
				switch mapValue.Kind() {
				case reflect.Pointer:
					if f.IsNil() {
						continue
					}
					ptr := mapValue.Interface()
					val := reflect.ValueOf(ptr)
					elem := val.Elem()
					if elem.Kind() == reflect.Struct {
						err := ApplyDefaults(ptr)
						if err != nil {
							return err
						}

						continue
					}
				case reflect.Struct:
					// go deeper
					ifx := mapValue.Addr().Interface()
					err := ApplyDefaults(ifx)
					if err != nil {
						return err
					}
				}
			}
		}

		tag := v.Tag
		defaultVal := tag.Get("default")
		if defaultVal == "" {
			continue
		}

		// apply the default if the field is the zero value
		if f.IsZero() {
			err := applyValue(f, defaultVal)
			if err != nil {
				return fmt.Errorf("parse error for field %s: %w", v.Name, err)
			}
		}
	}

	return nil
}

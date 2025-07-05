# kvstore
in-memory key value store for storing and accessing arbitrary data contexts

kvstore only stores JSON/YAML primitive types, storing / retrieving objects such as structs is not directly support without (de)serialization.  kvstore does not include any (de)serializers and thus support must be provided externally.

Example:
```
import (
  "github.com/frozengoats/kvstore"
)

// create an object
s := kvstore.NewStore()
s.Set("Bob", "person", "first_name")      // set value Bob at "person.first_name"
s.Set("Bork", "person", "last_name")      // set value Bork at "person.last_name"
s.Set("100 Sillyview Ave.", "address")    // set value "100 Sillyview Ave." at "address"

firstName := s.GetString("person", "first_name")    // retrieves the string value nested at "person.first_name" (if non-existant
                                                    // will be zero value)
```

while setting data is simple, taking the form of `Set(<value>, namespace...)`, data retrieval can be performed using a generic interface `Get` or any of the typed variants `GetInt`, `GetFloat`, `GetString`, etc.

Constructing from json
```
// create an object
m := map[string]any{}
json.Unmarshal(data, &m)

s, err := kvstore.FromMapping(m)
if err != nil {
  log.Fatal(err)
}

firstName := s.GetString("person", "first_name")
```

`kvstore` ensures that unsupported data cannot be stored, though certain data types will be recast at storage time, for instance `[]any` is supported, but `[]string`, etc. will be recreated as `[]any` prior to storage.

Overlaying stores
```
base := kvstore.NewStore()
ovl := kvstore.NewStore()

// populate both with data

final, err := base.Overlay(ovl)
if err != nil {
  log.Fatal(err)
}

// final now contains a new copy of the contents of base, with a copy of the contents of ovl on top.  the new copy is a deep copy.
```

Returning nested stores

```
// any nested mapping can be returned as a store, or array of stores (in the case of arrays of mappings)
s := kvstore.GetStore("x", "y", "z")

// or in the case of store arrays

s := kvstore.GetStoreArray("x", "y", "z")
```

Lookup by key as well as accessing arrays by index

```
// a data structure containing an array can be accessed by index
// if `s` is a store:

x := s.GetInt("x", "y", 3, "z")

// this would access the key x, then the subkey y, then the 4th array element (assuming an array), then the key z, on that array item.
// we can also use the following access notation to achieve the same result:

x := s.GetInt(ParseNamespaceString("x.y[3].z"))

// lastly, negative index values can be used to access values starting from the end of the array, whereby -1, is the last value,
// -2 is the second last, and so on

x := s.GetInt(ParseNamespaceString("x.y[-1]"))
```


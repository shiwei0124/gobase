// json_parser.go
package gobase

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

func UnmarshalJsonFromData(data []byte, object interface{}) error {
	if err := json.Unmarshal(data, object); err != nil {
		return err
	} else {
		return nil
	}
}

func UnmarshalJsonFromFile(file string, object interface{}) error {
	var err error
	var content []byte
	if content, err = ioutil.ReadFile(file); err != nil {
		return err
	} else {
		if err = UnmarshalJsonFromData(content, object); err != nil {
			return err
		}
	}
	return nil
}

func MarshalJsonToData(object interface{}) ([]byte, error) {
	if data, err := json.MarshalIndent(object, "", "    "); err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func MarshalJsonToFile(object interface{}, file string) error {
	if data, err := MarshalJsonToData(object); err != nil {
		return err
	} else {
		if err = ioutil.WriteFile(file, data, os.ModeExclusive); err != nil {
			return err
		} else {
			return nil
		}
	}
}

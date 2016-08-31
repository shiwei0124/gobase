// xml_parser.go
package gobase

import (
	"encoding/xml"
	"io/ioutil"
	"os"
)

func UnmarshalXmlFromData(data []byte, object interface{}) error {
	if err := xml.Unmarshal(data, object); err != nil {
		return err
	} else {
		return nil
	}
}

func UnmarshalXmlFromFile(file string, object interface{}) error {
	var err error
	var content []byte
	if content, err = ioutil.ReadFile(file); err != nil {
		return err
	} else {
		if err = UnmarshalXmlFromData(content, object); err != nil {
			return err
		}
	}
	return nil
}

func MarshalXmlToData(object interface{}) ([]byte, error) {
	if data, err := xml.MarshalIndent(object, "", "    "); err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func MarshalXmlToFile(object interface{}, file string) error {
	if data, err := MarshalXmlToData(object); err != nil {
		return err
	} else {
		if err = ioutil.WriteFile(file, data, os.ModeExclusive); err != nil {
			return err
		} else {
			return nil
		}
	}
}

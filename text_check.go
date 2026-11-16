package mwanachamaforms

import (
	"fmt"
	"reflect"

	"github.com/aosanya/mwanachama-backend-shared/plaintext"
)

func plainText(invalid error, label, text string) error {
	if err := plaintext.CheckText(label, text); err != nil {
		return fmt.Errorf("%w: %v", invalid, err)
	}
	return nil
}

func plainTextFields(invalid error, v any) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).PkgPath != "" || rv.Field(i).Kind() != reflect.String {
			continue
		}
		if err := plainText(invalid, columnName(rt.Field(i).Name), rv.Field(i).String()); err != nil {
			return err
		}
	}
	return nil
}

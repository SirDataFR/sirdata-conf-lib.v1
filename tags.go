package sirdataConf

import (
	"fmt"
	"reflect"
	"strings"
)

func GetTagKey(tagSpec func() (string, string), config interface{}, searched interface{}) (string, bool) {
	searchedValue := reflect.ValueOf(searched)
	if searchedValue.Kind() != reflect.Ptr {
		panic("Pointer required")
	}
	if searchedValue.Elem().Kind() == reflect.Struct {
		panic("Illegal structure value")
	}
	return getTagKeyRecursive(tagSpec, reflect.ValueOf(config), searchedValue.Pointer(), "")
}

func getTagKeyRecursive(tagSpec func() (string, string), value reflect.Value, ptr uintptr, tag string) (string, bool) {
	if value.Kind() == reflect.Ptr {
		return getTagKeyRecursive(tagSpec, value.Elem(), ptr, tag)
	}
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			structField := value.Type().Field(i)
			buildTag := buildTag(tagSpec, structField, tag)
			if elemTag, found := getTagKeyRecursive(tagSpec, value.Field(i), ptr, buildTag); found {
				return elemTag, true
			}
		}
	}
	if value.Addr().Pointer() == ptr {
		return tag, true
	}
	return "", false
}

func buildTag(tagSpec func() (string, string), field reflect.StructField, strPrefix string) string {
	tagName, tagSep := tagSpec()
	localTag := field.Tag.Get(tagName)
	localTag = strings.Split(localTag, ",")[0]
	if strPrefix == "" {
		return localTag
	}
	return fmt.Sprintf("%s%s%s", strPrefix, tagSep, localTag)
}

package client

import "github.com/bascanada/logviewer/pkg/ty"

type FieldRemapping struct{}

func (m FieldRemapping) RemapFieldSet(fields ty.UniSet[string]) ty.UniSet[string] {

	return fields
}

func (m FieldRemapping) RemapField(field ty.MI) ty.MI {

	return field
}

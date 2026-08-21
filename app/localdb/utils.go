package localdb

import "reflect"

// MergeNonZero overwrites fields in dst with fields in src iff src is not a zero value.
func MergeNonZero[T any](dst *T, src *T) {
	vDst := reflect.ValueOf(dst).Elem()
	vSrc := reflect.ValueOf(src).Elem()

	for i := 0; i < vDst.NumField(); i++ {
		dstField := vDst.Field(i)
		srcField := vSrc.Field(i)

		// Only set if the field is exported (CanSet) and the source is not a zero value
		if dstField.CanSet() && !srcField.IsZero() {
			dstField.Set(srcField)
		}
	}
}

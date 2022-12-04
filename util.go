package main

import "unsafe"

func ok1[T any](res T, err error) T {
	return res
}

func option1[T any](res T, err error) (T, bool) {
	return res, err != nil
}

func unwrap0(err error) {
	if err != nil {
		panic(err)
	}
}
func unwrap1[T any](res T, err error) T {
	unwrap0(err)
	return res
}

func unsafeByteSliceToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

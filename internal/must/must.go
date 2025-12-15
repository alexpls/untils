package must

func True(val bool) {
	if !val {
		panic("expected true, got false")
	}
}

func NoErrVal[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func NoErr(err error) {
	if err != nil {
		panic(err)
	}
}

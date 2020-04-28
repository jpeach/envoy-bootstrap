package must

// String ...
func String(s string, err error) string {
	if err != nil {
		panic(err.Error())
	}

	return s
}

package engine

// SetParitySpace installs the parity harness's space-object loader and
// checker (objects_parity_test.go, in package engine_test, which may
// import package objects).
func SetParitySpace(s pvSpace) { pvSpaceObjects = s }

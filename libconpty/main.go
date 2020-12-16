package libconpty

//import (
//	"C"
//	"fmt"
//)
//
//var (
//	conpty *ConPty
//)
//
////export Initialize
//func Initialize(cmd string, cols int16, rows int16) (err error) {
//	conpty, err = NewConPty(cmd, cols, rows)
//	return err
//}
//
////export Read
//func Read() (data []byte, err error) {
//	if conpty != nil {
//		return conpty.Read()
//	}
//	return nil, fmt.Errorf("ConPty has not been initialized")
//}
//
////export Write
//func Write(data []byte) (written int32, err error) {
//	if conpty != nil {
//		return conpty.Write(data)
//	}
//	return -1, fmt.Errorf("ConPty has not been initialized")
//}
//
//func main() {
//
//}

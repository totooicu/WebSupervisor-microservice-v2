package encryption
import (
	"crypto/md5"
	"fmt"
	"io"
)
func HashMD5(str string) string {
	hash := md5.New()
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

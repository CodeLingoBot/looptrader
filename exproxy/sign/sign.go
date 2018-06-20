package sign

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"
)

// signature msg
func Signature(msg string, secret []byte) string {
	bmsg := base64.StdEncoding.EncodeToString([]byte(msg))
	sh := hmac.New(sha1.New, secret)
	sh.Write([]byte(bmsg))
	return base64.StdEncoding.EncodeToString(sh.Sum(nil))
}

//
func MakeSignatureMessage(method, requrl string, timestamp int64, querys map[string]string) string {
	var (
		sigStr   string
		upMethod string = strings.ToUpper(method)
	)

	switch upMethod {
	case "GET":
		sigStr = fmt.Sprintf("%s%s%d", upMethod, requrl, timestamp)
	case "POST":
		sigStr = fmt.Sprintf("%s%s%d%s", upMethod, requrl, timestamp, SortMap(querys, "&"))
	}
	return sigStr
}

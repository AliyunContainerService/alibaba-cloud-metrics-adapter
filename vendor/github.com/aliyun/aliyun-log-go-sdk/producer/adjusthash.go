package producer

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func AdjustHash(shardhash string, buckets int) (string, error) {

	res := Md5ToBin(ToMd5(shardhash))
	x, err := BitCount(buckets)
	if err != nil {
		return "", err
	}
	tt := res[0:x]
	tt = FillZero(tt, 8)
	base, _ := strconv.ParseInt(tt, 2, 10)
	yy := strconv.FormatInt(base, 16)
	return FillZero(yy, 32), nil

}

// smilar as java Integer.bitCount
func BitCount(buckets int) (int, error) {
	bin := strconv.FormatInt(int64(buckets), 2)
	if strings.Contains(bin[1:], "1") || buckets <= 0 {
		return -1, errors.New(fmt.Sprintf("buckets must be a power of 2, got %v,and The parameter "+
			"buckets must be greater than or equal to 1 and less than or equal to 256.", buckets))
	}
	return strings.Count(bin, "0"), nil
}

func ToMd5(name string) string {
	h := md5.New()
	h.Write([]byte(name))
	cipherStr := h.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func Md5ToBin(md5 string) string {
	bArr, _ := hex.DecodeString(md5)
	res := ""
	for _, b := range bArr {
		res = fmt.Sprintf("%s%.8b", res, b)
	}
	return res
}

func FillZero(x string, n int) string {
	length := n - (strings.Count(x, "") - 1)
	for i := 0; i < length; i++ {
		x = x + "0"
	}
	return x
}

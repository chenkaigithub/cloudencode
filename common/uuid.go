package common

import (
	"encoding/base64"
	"fmt"
	"math/rand"

	"github.com/satori/go.uuid"
)

func NewId() string {
	id, _ := uuid.NewV4()
	src := id.Bytes()[:12]

	data := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(data, src)

	var buff []byte

	for _, item := range data {
		if (item >= 'A' && item <= 'Z') || (item >= 'a' && item <= 'z') || (item >= '0' && item <= '9') {
			buff = append(buff, item)
		}
	}

	diff := len(src) - len(buff)

	if diff > 7 {
		diff = 7
	}
	basenum := pow(10, diff)
	retRand := rand.Intn(basenum)
	if retRand == basenum {
		retRand = basenum - 1
	}

	var tailData []int
	for {
		number := retRand % 10
		tailData = append(tailData, number)

		retRand = retRand / 10
		if retRand == 0 {
			break
		}
	}

	retStr := string(buff)
	for _, item := range tailData {
		retStr = fmt.Sprintf("%s%d", retStr, item)
	}

	return retStr
}

func pow(x, n int) int {
	ret := 1 // 结果初始为0次方的值，整数0次方为1。如果是矩阵，则为单元矩阵。
	for n != 0 {
		if n%2 != 0 {
			ret = ret * x
		}
		n /= 2
		x = x * x
	}
	return ret
}

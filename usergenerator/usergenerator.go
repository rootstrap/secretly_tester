package usergenerator

import (
	crand "crypto/rand"
	"errors"
	"math/big"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

type UserGenerator struct {
	rand       *rand.Rand
	offset     int32
	percentNew int32
}

func NewUserGenerator(offset int32, percentNew int32) (ug *UserGenerator, err error) {
	ug = new(UserGenerator)
	rand, err := newRand()
	if err != nil {
		return
	}
	ug.rand = rand
	ug.offset = offset
	if percentNew < 0 || percentNew > 100 {
		err = errors.New("percent new users must be between 0 and 100")
		return
	}
	ug.percentNew = percentNew
	return
}

func (ug *UserGenerator) Gen() (string, bool) {
	if ug.shouldMakeRandomUser() {
		return "testfan" + RandomString(20), true
	}
	return "testfan" + strconv.Itoa(int(atomic.AddInt32(&ug.offset, 1))-1), false
}

func (ug *UserGenerator) GetExisting(n int) []string {
	users := make([]string, n)
	for i := 0; i < n; i++ {
		users[i] = "testfan" + strconv.Itoa(int(ug.offset)+i)
	}
	return users
}

func (ug *UserGenerator) shouldMakeRandomUser() bool {
	return ug.rand.Int31n(100) < ug.percentNew
}

func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func newRand() (*rand.Rand, error) {
	absSeed, err := crand.Int(crand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		return nil, err
	}
	seed := new(big.Int).Sub(absSeed, new(big.Int).Lsh(big.NewInt(1), 63)).Int64()
	return rand.New(rand.NewSource(seed)), nil
}

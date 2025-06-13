package main

import (
	"math/rand"
	"sync"
	"time"

	"github.com/4r7hur0/PBL-2/schemas"
)

var enterprises []schemas.Enterprises
var mu sync.Mutex // Mutex to handle concurrent access to the list

func ChooseTwoRandomCities() (string, string) {
	mu.Lock()
	defer mu.Unlock()

	if len(enterprises) < 2 {
		return "", ""
	}

	rand.Seed(time.Now().UnixNano())
	indx1 := rand.Intn(len(enterprises))
	indx2 := rand.Intn(len(enterprises))

	for indx1 == indx2 {
		indx2 = rand.Intn(len(enterprises))
	}

	return enterprises[indx1].City, enterprises[indx2].City
}

func chooseRandomEnterprise() *schemas.Enterprises {
	mu.Lock()
	defer mu.Unlock()

	if len(enterprises) == 0 {
		return nil
	}

	rand.Seed(time.Now().UnixNano())
	indx := rand.Intn(len(enterprises))
	return &enterprises[indx]
}

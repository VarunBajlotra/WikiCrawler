package main

import (
	"fmt"
	"strings"
	"strconv"
	"encoding/json"
	"github.com/jmhodges/levigo"
	"sort"
)

func main() {

	// Open the db
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3<<30))
	opts.SetCreateIfMissing(true)
	db, _ := levigo.Open("dictionary", opts)

	flag := "yes"

	for flag == "yes" {

		fmt.Println("Please enter the keyword you want to search: ")
		var keyword string
		fmt.Scanln(&keyword)

		// Reading from the db
		ro := levigo.NewReadOptions()
		data, _ := db.Get(ro, []byte(keyword))

		// Converting stream of bytes to array of strings(urls)
		strArray := []string{}
		json.Unmarshal(data, &strArray)

		sort.Slice(strArray, func(i1, i2 int) bool {
			pair1 := strings.Split(strArray[i1], ",")
			count1, _ := strconv.Atoi(pair1[1])
			pair2 := strings.Split(strArray[i2], ",")
			count2, _ := strconv.Atoi(pair2[1])
			return count1 > count2
		})

		count := len(strArray)

		fmt.Println("\nThe links are listed along with the frequency of the keyword in the corresponding URL.\n ")
		for i := 0; i < count; i++ {

			// Extracting the url and frequency of occurence
			str := strArray[i]
			pair := strings.Split(str, ",")
			uri := pair[0]
			count, _ := strconv.Atoi(pair[1])
			fmt.Println(uri, count)
		}
		fmt.Println("\nFetched", count, "urls for your keyword.\n ")

		fmt.Println("\nDo you wish to enter another keyword? (yes/no)")
		fmt.Scanln(&flag)
	}

	defer db.Close()
}

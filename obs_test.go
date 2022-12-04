package main

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio-go/v7/pkg/signer"
)

func TestObsSignerV2(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/test/mk/603d83c0-5083-44b0-87cb-7030ef28c43f.jpg", nil)
	if err != nil {
		t.Fail()
	}

	exp := strconv.FormatInt(int64(^uint64(0)/2), 10) // ~250years
	req.Header.Set("Expires", exp)
	req.URL.RawQuery = ""
	reqVal := signer.PreSignV2(*req, "asd", "asdasd", 0, true)
	query := reqVal.URL.Query()
	query.Set("Expires", exp)
	reqVal.URL.RawQuery = s3utils.QueryEncode(query)

	fmt.Println(reqVal.URL)

}

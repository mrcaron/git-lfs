package main

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/github/git-lfs/tools"
)

// "upload" - all missing
func uploadAllMissing(oidsExist, oidsMissing []TestObject) error {
	retobjs, err := callBatchApi("upload", oidsMissing)

	if err != nil {
		return err
	}

	if len(retobjs) != len(oidsMissing) {
		return fmt.Errorf("Incorrect number of returned objects, expected %d, got %d", len(oidsMissing), len(retobjs))
	}

	var errbuf bytes.Buffer
	for _, o := range retobjs {
		_, ok := o.Rel("upload")
		if !ok {
			errbuf.WriteString(fmt.Sprintf("Missing upload link for %s\n", o.Oid))
		}
		// verify link is optional so don't check
	}

	if errbuf.Len() > 0 {
		return errors.New(errbuf.String())
	}

	return nil
}

// "upload" - all present
func uploadAllExists(oidsExist, oidsMissing []TestObject) error {
	retobjs, err := callBatchApi("upload", oidsExist)

	if err != nil {
		return err
	}

	if len(retobjs) != len(oidsExist) {
		return fmt.Errorf("Incorrect number of returned objects, expected %d, got %d", len(oidsExist), len(retobjs))
	}

	var errbuf bytes.Buffer
	for _, o := range retobjs {
		link, ok := o.Rel("upload")
		if ok {
			errbuf.WriteString(fmt.Sprintf("Upload link should not exist for %s, was %s\n", o.Oid, link))
		}
	}

	if errbuf.Len() > 0 {
		return errors.New(errbuf.String())
	}

	return nil
}

// "upload" - mix of missing & present
func uploadMixed(oidsExist, oidsMissing []TestObject) error {

	existSet := tools.NewStringSetWithCapacity(len(oidsExist))
	for _, o := range oidsExist {
		existSet.Add(o.Oid)
	}
	missingSet := tools.NewStringSetWithCapacity(len(oidsMissing))
	for _, o := range oidsMissing {
		missingSet.Add(o.Oid)
	}

	calloids := interleaveTestData(oidsExist, oidsMissing)
	retobjs, err := callBatchApi("upload", calloids)

	if err != nil {
		return err
	}

	count := len(oidsExist) + len(oidsMissing)
	if len(retobjs) != count {
		return fmt.Errorf("Incorrect number of returned objects, expected %d, got %d", count, len(retobjs))
	}

	var errbuf bytes.Buffer
	for _, o := range retobjs {
		link, ok := o.Rel("upload")
		if existSet.Contains(o.Oid) {
			if ok {
				errbuf.WriteString(fmt.Sprintf("Upload link should not exist for %s, was %s\n", o.Oid, link))
			}
		}
		if missingSet.Contains(o.Oid) && !ok {
			errbuf.WriteString(fmt.Sprintf("Missing upload link for %s\n", o.Oid))
		}

	}

	if errbuf.Len() > 0 {
		return errors.New(errbuf.String())
	}

	return nil

}

func uploadEdgeCases(oidsExist, oidsMissing []TestObject) error {
	errorCases := make([]TestObject, 0, 5)
	errorCodeMap := make(map[string]int, 5)
	errorReasonMap := make(map[string]string, 5)
	validCases := make([]TestObject, 0, 1)
	validReasonMap := make(map[string]string, 5)

	// Invalid SHAs - code 422
	// Too short
	sha := "a345cde"
	errorCases = append(errorCases, TestObject{Oid: sha, Size: 99})
	errorCodeMap[sha] = 422
	errorReasonMap[sha] = "SHA is too short"
	// Too long
	sha = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	errorCases = append(errorCases, TestObject{Oid: sha, Size: 99})
	errorCodeMap[sha] = 422
	errorReasonMap[sha] = "SHA is too long"
	// Invalid characters -----!---------------------------------!
	sha = "bf3e3e2af9366a3b704ax0c31de5afa64193ebabffde2091936ad2G7510bc03a"
	errorCases = append(errorCases, TestObject{Oid: sha, Size: 99})
	errorCodeMap[sha] = 422
	errorReasonMap[sha] = "SHA contains invalid characters"

	// Invalid size - code 422
	sha = "e3bf3e2af9366a3b704af0c31de5afa64193ebabffde2091936ad237510bc03a"
	errorCases = append(errorCases, TestObject{Oid: sha, Size: -1})
	errorCodeMap[sha] = 422
	errorReasonMap[sha] = "Negative size"
	sha = "d2983e2af9366a3b704af0c31de5afa64193ebabffde2091936ad237510bc03a"
	errorCases = append(errorCases, TestObject{Oid: sha, Size: -125})
	errorCodeMap[sha] = 422
	errorReasonMap[sha] = "Negative size"

	// Zero size - should be allowed
	sha = "159f6ac723b9023b704af0c31de5afa64193ebabffde2091936ad237510bc03a"
	validCases = append(validCases, TestObject{Oid: sha, Size: 0})
	validReasonMap[sha] = "Zero size should be allowed"

	calloids := interleaveTestData(errorCases, validCases)
	retobjs, err := callBatchApi("upload", calloids)

	if err != nil {
		return err
	}

	count := len(errorCases) + len(validCases)
	if len(retobjs) != count {
		return fmt.Errorf("Incorrect number of returned objects, expected %d, got %d", count, len(retobjs))
	}

	var errbuf bytes.Buffer
	for _, o := range retobjs {
		link, ok := o.Rel("upload")
		if code, iserror := errorCodeMap[o.Oid]; iserror {
			reason, _ := errorReasonMap[o.Oid]
			if ok {
				errbuf.WriteString(fmt.Sprintf("Upload link should not exist for %s, was %s, reason %s\n", o.Oid, link, reason))
			}
			if o.Error == nil {
				errbuf.WriteString(fmt.Sprintf("Upload should include an error for invalid object %s, reason %s", o.Oid, reason))
			} else if o.Error.Code != code {
				errbuf.WriteString(fmt.Sprintf("Upload error code for missing object %s should be %d, got %d, reason %s\n", o.Oid, code, o.Error.Code, reason))
			}

		}
		if reason, reasonok := validReasonMap[o.Oid]; reasonok {
			if !ok {
				errbuf.WriteString(fmt.Sprintf("Missing upload link for %s, should be present because %s\n", o.Oid, reason))
			}
		}

	}

	if errbuf.Len() > 0 {
		return errors.New(errbuf.String())
	}

	return nil

}

func init() {
	addTest("Test upload: all missing", uploadAllMissing)
	addTest("Test upload: all present", uploadAllExists)
	addTest("Test upload: mixed", uploadMixed)
	addTest("Test upload: edge cases", uploadEdgeCases)
}

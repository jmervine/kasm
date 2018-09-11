package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var (
	// If true, will fail if there's no secretmanager key and no default.
	fail bool

	region string
	svc    *secretsmanager.SecretsManager

	pattern = regexp.MustCompile("\"secretmanager:[A-Za-z0-9\\/\\-\\_\\|]+\"")
	encode  = false
	keys    = new([]string)
)

func init() {
	// Attempt to find REGION for AWS Libs, let the lib error if not found.
	region = os.Getenv("REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	svc = secretsmanager.New(sess)

	flag.BoolVar(&fail, "x", false, "Fail if the key is not found and there is no default set.")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n%s [options] FILE:\n", os.Args[0])
		flag.PrintDefaults()
	}

}

func main() {
	flag.Parse()

	path := flag.Arg(0)
	if r, e := findAndReplace(path); e != nil {
		fmt.Println(e)
	} else {
		print(r)
	}
}

func findAndReplace(path string) (string, error) {
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	file := string(dat)

	if strings.Contains(file, "kind: Secret\n") {
		encode = true
	}

	matches := pattern.FindAllString(file, -1)

	for _, match := range matches {

		key := strings.TrimPrefix(match, "\"secretmanager:")
		key = strings.TrimSuffix(key, "\"")
		parts := strings.SplitN(key, "|", 2)
		key = parts[0]

		var backup string
		if len(parts) == 2 {
			backup = parts[1]
		}

		val, err := getSecret(key)
		if err != nil {
			val = backup
		}

		if encode {
			val = base64.StdEncoding.EncodeToString([]byte(val))
		}

		file = strings.Replace(file, match, val, -1)
	}

	return file, nil
}

func getSecret(k string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(k),
		VersionStage: aws.String("AWSCURRENT"),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeResourceNotFoundException:
				return "", fmt.Errorf("%s %s", secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			case secretsmanager.ErrCodeInvalidParameterException:
				return "", fmt.Errorf("%s %s", secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
			case secretsmanager.ErrCodeInvalidRequestException:
				return "", fmt.Errorf("%s %s", secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
			case secretsmanager.ErrCodeDecryptionFailure:
				return "", fmt.Errorf("%s %s", secretsmanager.ErrCodeDecryptionFailure, aerr.Error())
			case secretsmanager.ErrCodeInternalServiceError:
				return "", fmt.Errorf("%s %s", secretsmanager.ErrCodeInternalServiceError, aerr.Error())
			default:
				return "", fmt.Errorf("%s", aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			return "", fmt.Errorf("%s", aerr.Error())
		}
		return "", err
	}

	return *result.SecretString, nil
}

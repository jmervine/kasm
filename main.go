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

const VERSION = "0.0.1"

var (
	// If true, will fail if there's no secretmanager key and no default.
	fail, ver bool

	region string
	svc    *secretsmanager.SecretsManager

	pattern = regexp.MustCompile(`{{\s?secretmanager "([a-zA-Z0-9\\-\\_\\/]+)"(\s"([a-zA-Z0-9\\-\\_\\/]*)")?\s?}}`)
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
	flag.BoolVar(&ver, "v", false, "Display version.")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n%s [options] FILE:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

}

func main() {
	flag.Parse()

	if ver {
		fmt.Printf("%s %s\n", os.Args[0], VERSION)
		os.Exit(0)
	}

	path := flag.Arg(0)
	if r, e := apply(path); e != nil {
		fmt.Println(e)
	} else {
		fmt.Print(r)
	}
}

func findAndReplace(file string) string {
	matches := pattern.FindAllStringSubmatch(file, -1)
	for _, match := range matches {
		orig := match[0]
		key := match[1]
		bak := match[3]

		val, err := getSecret(key)
		if err != nil && fail {
			fmt.Println(err)
			os.Exit(1)
		} else if err != nil {
			val = bak
		}

		if encode {
			val = base64.StdEncoding.EncodeToString([]byte(val))
		}

		file = strings.Replace(file, orig, val, -1)
	}

	return file
}

func apply(path string) (string, error) {
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	file := string(dat)

	if strings.Contains(file, "kind: Secret\n") {
		encode = true
	}

	return findAndReplace(file), nil
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

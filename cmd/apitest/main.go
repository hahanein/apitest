// Copyright Programmfabrik GmbH
// All Rights Reserved
package main

import (
	"fmt"
	"io/ioutil"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/report"
	log "github.com/sirupsen/logrus"

	"path/filepath"

	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reportFormat, reportFile                                           string
	logNetwork, logDatastore, logVerbose, logTimeStamp, limit, logCurl bool
	rootDirectorys, singleTests                                        []string
)

func init() {
	//Configure all the flags that fylr-apitest offers
	TestCMD.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./apitest.yml", "config file")

	TestCMD.PersistentFlags().StringSliceVarP(
		&rootDirectorys, "directory", "d", []string{"."},
		"path to directory containing the tests.")

	TestCMD.PersistentFlags().StringSliceVarP(
		&singleTests, "single", "s", []string{},
		"path to a single manifest. Runs only that specified testsuite")

	TestCMD.PersistentFlags().BoolVarP(
		&logNetwork, "log-network", "n", false,
		`log all network traffic to console`)
	TestCMD.PersistentFlags().BoolVarP(
		&logVerbose, "log-verbose", "v", false,
		`log datastore operations and information about repeating request to console`)
	TestCMD.PersistentFlags().BoolVar(
		&logDatastore, "log-datastore", false,
		`log datastore operations`)

	TestCMD.PersistentFlags().BoolVarP(
		&logTimeStamp, "log-timestamp", "t", false,
		`log full timestamp into console`)

	TestCMD.PersistentFlags().StringVar(
		&reportFile, "report-file", "",
		"Defines where the log statements should be saved.")

	TestCMD.PersistentFlags().StringVar(
		&reportFormat, "report-format", "",
		"Defines how the report statements should be saved. [junit/json]")

	TestCMD.PersistentFlags().BoolVarP(
		&limit, "limit", "l", false,
		"Limit the lines of request log output. Set limits in apitest.yml")

	TestCMD.PersistentFlags().BoolVar(
		&logCurl, "curl-bash", false,
		"Log network output as bash curl command")

	//Bind the flags to overwrite the yml config if they are set
	viper.BindPFlag("apitest.report.file", TestCMD.PersistentFlags().Lookup("report-file"))
	viper.BindPFlag("apitest.report.format", TestCMD.PersistentFlags().Lookup("report-format"))

}

var TestCMD = &cobra.Command{
	Args:             cobra.MaximumNArgs(0),
	PersistentPreRun: setup,
	Use:              "fylr apitest",
	Short:            "flyr Apitester lets you define API tests on the go",
	Long:             `A fast and flexible API testing tool. Helping you to define API tests on the go`,
	Run:              runApiTests,
}

func main() {
	err := TestCMD.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var cfgFile string

func setup(ccmd *cobra.Command, args []string) {
	//Load yml config
	LoadConfig(cfgFile)

	//Set log verbosity to trace
	log.SetLevel(log.TraceLevel)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: logTimeStamp,
	})

}

func runApiTests(cmd *cobra.Command, args []string) {

	//Check if paths are valid

	for _, rootDirectory := range rootDirectorys {
		if _, err := os.Stat(rootDirectory); rootDirectory != "." && os.IsNotExist(err) {
			log.Fatalf("The path '%s' for the test folders is not valid", rootDirectory)
		}
	}
	for _, singleTest := range singleTests {
		if _, err := os.Stat(singleTest); singleTest != "" && os.IsNotExist(err) {
			log.Fatalf("The path '%s' for the single test is not valid", singleTest)
		}
	}

	serverUrl := FylrConfig.Apitest.Server
	reportFormat = FylrConfig.Apitest.Report.Format
	reportFile = FylrConfig.Apitest.Report.File

	//Save the config into TestToolConfig
	testToolConfig, err := NewTestToolConfig(serverUrl, rootDirectorys, logNetwork, logVerbose)
	if err != nil {
		log.Fatal(err)
	}

	//Actually run the tests
	//Run test function
	runSingleTest := func(manifestPath string, r *report.ReportElement) (success bool) {
		store := datastore.NewStore(logVerbose || logDatastore)
		for k, v := range FylrConfig.Apitest.StoreInit {
			err := store.Set(k, v)
			if err != nil {
				log.Errorf("Could not add init value for datastore Key: '%s', Value: '%v'. %s", k, v, err)
			}
		}

		suite, err := NewTestSuite(
			testToolConfig,
			manifestPath,
			r,
			store,
			0,
		)
		if err != nil {
			log.Fatal(err)
		}

		return suite.Run()
	}

	r := report.NewReport()

	//Decide if run only one test
	if len(singleTests) > 0 {
		for _, singleTest := range singleTests {
			absManifestPath, _ := filepath.Abs(singleTest)
			c := r.Root().NewChild(singleTest)
			c.Leave(runSingleTest(absManifestPath, c))
		}
	} else {
		for _, singlerootDirectory := range testToolConfig.TestDirectories {
			manifestPath := filepath.Join(singlerootDirectory, "manifest.json")
			absManifestPath, _ := filepath.Abs(manifestPath)
			c := r.Root().NewChild(manifestPath)
			c.Leave(runSingleTest(absManifestPath, c))
		}
	}

	//Create report
	if reportFile != "" {
		var parsingFunction func(baseResult *report.ReportElement) []byte
		switch reportFormat {
		case "junit":
			parsingFunction = report.ParseJUnitResult
		case "json":
			parsingFunction = report.ParseJSONResult
		default:
			log.Errorf(
				"Given report format '%s' not supported. Saving report '%s' as json",
				reportFormat,
				reportFile)

			parsingFunction = report.ParseJSONResult
		}

		err = ioutil.WriteFile(reportFile, r.GetTestResult(parsingFunction), 0644)
		if err != nil {
			log.Errorf("Could not save report into file: %s", err)
		}
	}

	if r.DidFail() {
		os.Exit(1)
	}
}
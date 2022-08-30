package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"code.cloudfoundry.org/cli/plugin"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/olekukonko/tablewriter"
	"github.com/remeh/sizedwaitgroup"
	pb "gopkg.in/cheggaaa/pb.v1"
)

type MemShame struct {
	org   string
	space string
	hr    bool
}

type appStat struct {
	Name         string
	GUID         string
	Space        string
	Instances    int
	MemoryAlloc  int
	AvgMemoryUse int
	Ratio        float64
}

func (s *appStat) toValueList() []string {
	return []string{s.Name, s.Space, fmt.Sprintf("%d", s.MemoryAlloc), fmt.Sprintf("%d", s.AvgMemoryUse), fmt.Sprintf("%f", s.Ratio)}
}

type byRatio []appStat

func (a byRatio) Len() int           { return len(a) }
func (a byRatio) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRatio) Less(i, j int) bool { return a[j].Ratio < a[i].Ratio }

func main() {
	plugin.Start(new(MemShame))
}

func (memShame *MemShame) Run(cliConnection plugin.CliConnection, args []string) {
	//define and Parse args
	memShameFlagSet := flag.NewFlagSet("memshame", flag.ExitOnError)
	org := memShameFlagSet.String("org", "", "set org to review")
	space := memShameFlagSet.String("space", "", "set space to review (requires -org)")
	hr := memShameFlagSet.Bool("hr", false, "set memory to human readable in MBs")

	var appStats []appStat
	err := memShameFlagSet.Parse(args[1:])
	if err != nil {
		panic(err)
	}

	if *org == "" && *space != "" {
		fmt.Printf("Please set -org when using -space\n")
	}

	//Establish connection to capi
	apiEndpoint, err := cliConnection.ApiEndpoint()
	if err != nil {
		panic(err)
	}

	token, err := cliConnection.AccessToken()
	if err != nil {
		panic(err)
	}

	cfToken := token[7:len(token)]

	cfUser, err := cliConnection.Username()
	if err != nil {
		panic(err)
	}

	cfconfig := &cfclient.Config{
		ApiAddress:        apiEndpoint,
		Username:          cfUser,
		Token:             cfToken,
		SkipSslValidation: true, //TODO: make this configurable
		ClientSecret:      "",
		ClientID:          "cf",
	}

	client, err := cfclient.NewClient(cfconfig)
	if err != nil {
		panic(err)
	}

	apps, err := client.ListApps() //Error requesting apps: cfclient: error (1000): CF-InvalidAuthToken
	if err != nil {
		panic(err)
	}

	bar := pb.StartNew(len(apps))

	wg := sizedwaitgroup.New(20)
	for _, app := range apps {
		wg.Add()
		go func(cfApp cfclient.App, pb *pb.ProgressBar) {
			defer wg.Done()
			stats, err := client.GetAppStats(cfApp.Guid)

			pb.Increment()

			if err != nil {
				return
			}

			if stats["0"].State != "RUNNING" {
				return
			}
			memAlloc := stats["0"].Stats.MemQuota

			var totalUsage int
			for _, stat := range stats {
				totalUsage += stat.Stats.Usage.Mem
			}

			if *hr == true {
				memAlloc = (memAlloc / 1024 / 1024)
				totalUsage = (totalUsage / 1024 / 1024)
			}

			stat := appStat{
				Name:         cfApp.Name,
				GUID:         cfApp.Guid,
				Instances:    cfApp.Instances,
				MemoryAlloc:  memAlloc,
				Space:        cfApp.SpaceGuid,
				AvgMemoryUse: totalUsage / len(stats),
				Ratio:        float64(memAlloc) / float64(totalUsage/len(stats)),
			}

			appStats = append(appStats, stat)
		}(app, bar)
	}

	wg.Wait()

	bar.FinishPrint("Done!")

	sort.Sort(byRatio(appStats))

	table := tablewriter.NewWriter(os.Stdout)
	if *hr == true {
		table.SetHeader([]string{"Name", "Space", "Alloc (MBs)", "AvgUse (MBs)", "Ratio"})
	} else {
		table.SetHeader([]string{"Name", "Space", "Alloc", "AvgUse", "Ratio"})
	}

	for _, v := range appStats {
		table.Append(v.toValueList())
	}

	table.Render()
	//Get GUID list of orgs filtering by org flag
	//Get GUID list of space from each org flitering by org/space flag
	//get GUID list of stacks
	//Get list of apps in each org & space with name and stack details
	//display list of apps (Org,Space, app, count of stack X, count of stack y, count of stack z) in tabular format

}

func (memShame *MemShame) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "MemShame",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
			Build: 1,
		},
		Commands: []plugin.Command{
			{
				Name:     "Memory Hall of Shame",
				Alias:    "memshame",
				HelpText: "Reviews memory usages by  orgs and space. To obtain more information use --help",
				UsageDetails: plugin.Usage{
					Usage: "memshame - list memory in use by org and space.\n   cf memshame [-org] [-space]",
					Options: map[string]string{
						"org":   "Specify the org to report",
						"space": "Specify the space to report (requires -org)",
						"hr":    "Output memory in human readable format",
					},
				},
			},
		},
	}
}

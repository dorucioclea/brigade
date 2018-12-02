package commands

import (
	"fmt"
	"io"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/Azure/brigade/pkg/brigade"
	"github.com/Azure/brigade/pkg/storage/kube"
)

const buildListUsage = `List all installed builds.

Print a list of the current builds starting from latest (in creation time) to oldest. By default it will print all the builds, use --count to get a subset of them.
`

var buildListCount int

func init() {
	build.AddCommand(buildList)
	buildList.Flags().IntVar(&buildListCount, "count", 0, "The maximum number of builds to return. 0 for all")
}

var buildList = &cobra.Command{
	Use:   "list [project]",
	Short: "list builds",
	Long:  buildListUsage,
	RunE: func(cmd *cobra.Command, args []string) error {
		proj := ""
		if len(args) > 0 {
			proj = args[0]
		}

		c, err := kubeClient()
		if err != nil {
			return err
		}

		bls, err := getBuilds(proj, c, buildListCount)
		if err != nil {
			return err
		}

		listBuilds(bls, cmd.OutOrStdout())
		return nil
	},
}

func listBuilds(bs []*buildForStdout, out io.Writer) {
	table := uitable.New()
	table.AddRow("ID", "TYPE", "PROVIDER", "PROJECT", "STATUS", "AGE")
	for _, b := range bs {
		table.AddRow(b.ID, b.Type, b.Provider, b.ProjectID, b.status, b.since)
	}
	fmt.Fprintln(out, table)
}

func getBuilds(project string, client kubernetes.Interface, count int) ([]*buildForStdout, error) {

	store := kube.New(client, globalNamespace)

	var builds []*brigade.Build
	var err error
	if project == "" {
		builds, err = store.GetBuilds()
		if err != nil {
			return nil, err
		}
	} else {
		proj, err := store.GetProject(project)
		if err != nil {
			return nil, err
		}

		builds, err = store.GetProjectBuilds(proj)
		if err != nil {
			return nil, err
		}
	}

	var bfss []*buildForStdout
	for i := len(builds) - 1; i >= 0; i-- {
		if count > 0 && len(builds)-i-1 >= count {
			break
		}

		b := builds[i]
		bfs := &buildForStdout{Build: builds[i]}

		bfs.status = "???"
		bfs.since = "???"
		if b.Worker != nil {
			bfs.status = b.Worker.Status.String()
			if b.Worker.Status == brigade.JobSucceeded || b.Worker.Status == brigade.JobFailed {
				bfs.since = duration.ShortHumanDuration(time.Since(b.Worker.EndTime))
			}
		}
		bfss = append(bfss, bfs)
	}

	return bfss, nil
}

type buildForStdout struct {
	*brigade.Build
	status string
	since  string
}

package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/nanovms/ops/qemu"
	"github.com/spf13/cobra"

	api "github.com/nanovms/ops/lepton"
)

// ComposeCommands provides support for running binary with nanos
func ComposeCommands() *cobra.Command {

	var cmdCompose = &cobra.Command{
		Use:       "compose",
		Short:     "orchestrate multiple unikernels",
		ValidArgs: []string{"up", "down"},
		Args:      cobra.OnlyValidArgs,
	}

	cmdCompose.AddCommand(composeUpCommand())
	cmdCompose.AddCommand(composeDownCommand())

	return cmdCompose
}

func composeDownCommand() *cobra.Command {
	var cmdDownCompose = &cobra.Command{
		Use:   "down",
		Short: "spin unikernels down",
		Run:   composeDownCommandHandler,
	}

	return cmdDownCompose
}

func composeUpCommand() *cobra.Command {
	var cmdUpCompose = &cobra.Command{
		Use:   "up",
		Short: "spin unikernels up",
		Run:   composeUpCommandHandler,
	}

	cmdUpCompose.PersistentFlags().StringP("compose-file", "f", "", "compose file (default: cwd)")
	return cmdUpCompose
}

func composeDownCommandHandler(cmd *cobra.Command, args []string) {
	if runtime.GOOS == "darwin" && qemu.OPSD == "" {
		fmt.Println("this command is only enabled if you have OPSD compiled in.")
		os.Exit(1)
	}

	c := api.NewConfig()

	p, ctx, err := getProviderAndContext(c, "onprem")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	// this looks like it's killing everything when it should only be
	// killing the set desired.
	instances, err := p.GetInstances(ctx)
	if err != nil {
		fmt.Println(err)
	}

	// get bridge of one of these instances..

	for i := 0; i < len(instances); i++ {
		err = p.DeleteInstance(ctx, instances[i].Name)
		if err != nil {
			exitWithError(err.Error())
		}
	}

	if runtime.GOOS == "linux" {
		body := getComposeContents("")

		h := sha1.New()
		h.Write([]byte(body))
		sha := hex.EncodeToString(h.Sum(nil))

		opshome := api.GetOpsHome()
		composes := path.Join(opshome, "composes")

		body, err := os.ReadFile(composes + "/" + sha)
		if err != nil {
			fmt.Println(err)
		}

		brName := string(body)

		// need to tune down the taps here too...
		removeBridge(brName)
	}
}

func composeUpCommandHandler(cmd *cobra.Command, args []string) {
	if runtime.GOOS == "darwin" && qemu.OPSD == "" {
		fmt.Println("this command is only enabled if you have OPSD compiled in.")
		os.Exit(1)
	}

	flags := cmd.Flags()
	globalFlags := NewGlobalCommandFlags(flags)

	composeFile, _ := cmd.Flags().GetString("compose-file")

	c := api.NewConfig()
	mergeContainer := NewMergeConfigContainer(globalFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	if c.Kernel == "" {
		version, err := getCurrentVersion()
		if err != nil {
			fmt.Println(err)
		}
		version = setKernelVersion(version)

		c.Kernel = getKernelVersion(version)
	}

	com := Compose{
		config: c,
	}
	com.UP(composeFile)

}

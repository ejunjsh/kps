package main

import (
	"context"
	"encoding/json"
	"fmt"
	dockerterm "github.com/docker/docker/pkg/term"
	"github.com/ejunjsh/kps/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/remotecommand"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
	"net/url"
	"os"
)

func main()  {


	var image,containerName,namespace string

	cmd := &cobra.Command{
		Use:                   "",
		DisableFlagsInUseLine: true,
		Short:                 "",
		Long:                  "",
		Example:               "",
		Version:               "0.1",
		Run: func(c *cobra.Command, args []string) {
			podName := args[0]
			command := args[1:]
			if len(command) < 1 {
				command = []string{"bash"}
			}

			kcli, err:= client.NewKubeClient()
			if err!=nil{
				cmdutil.CheckErr(err)
			}

			ctx,cancel:=context.WithCancel(context.Background())
			defer cancel()
			pod ,err:= kcli.GetPodByName(ctx,namespace,podName)
			if err!=nil{
				cmdutil.CheckErr(err)
			}

			if len(containerName) == 0 {
				if len(pod.Spec.Containers) > 1 {
					usageString := fmt.Sprintf("Defaulting container name to %s.", pod.Spec.Containers[0].Name)
					fmt.Printf( "%s\n\r", usageString)
				}
				containerName = pod.Spec.Containers[0].Name
			}

			containerUri, err:=kcli.GetContainerIDByName(pod,containerName)
			if err!=nil{
				cmdutil.CheckErr(err)
			}

			remoteUrl:=kcli.GetControllerUrl(pod)
			params := url.Values{}
			params.Add("image", image)
			params.Add("containerUri", containerUri)
			commandBytes, err := json.Marshal(command)
			if err != nil {
				cmdutil.CheckErr(err)
			}
			params.Add("cmd",string(commandBytes))

			remoteUrl.RawQuery = params.Encode()

			t := setupTTY()
			var sizeQueue remotecommand.TerminalSizeQueue
			if t.Raw {
				sizeQueue = t.MonitorSize(t.GetSize())
			}

			err = kcli.RemoteExecute("POST", remoteUrl, os.Stdin, os.Stdout, os.Stderr, t.Raw, sizeQueue)
			if err != nil {
				cmdutil.CheckErr(err)
			}
		},
	}


	cmd.Flags().StringVar(&image, "image", "", "")
	cmd.Flags().StringVarP(&containerName, "container", "c","","")
	cmd.Flags().StringVarP(&namespace, "namespace", "n","","")


	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func  setupTTY() term.TTY {
	t := term.TTY{
		Out: os.Stdout,
	}
	t.In = os.Stdin
	t.Raw = true
	if !t.IsTerminalIn() {
		return t
	}
	stdin, stdout, _ := dockerterm.StdStreams()
	t.In = stdin
	t.Out = stdout
	return t
}

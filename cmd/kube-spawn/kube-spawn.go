/*
Copyright 2017 Kinvolk GmbH

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const k8sStableVersion string = "1.7.0"
const defaultRuntime string = "docker"
const kubeSpawnDirDefault string = "/var/lib/kube-spawn"

var (
	cmdKubeSpawn = &cobra.Command{
		Use:   "kube-spawn",
		Short: "kube-spawn is a tool for creating a multi-node dev Kubernetes cluster",
		Long:  "kube-spawn is a tool for creating a multi-node dev Kubernetes cluster, by using the local source code and systemd-nspawn containers",
		Run: func(cmd *cobra.Command, args []string) {
			if printVersion {
				fmt.Printf("kube-spawn %s\n", version)
				os.Exit(0)
			}
			if err := cmd.Usage(); err != nil {
				log.Fatal(err)
			}
		},
	}

	version      string
	k8srelease   string
	k8sruntime   string
	printVersion bool
	kubeSpawnDir string
)

func init() {
	cmdKubeSpawn.Flags().BoolVarP(&printVersion, "version", "V", false, "Output version information")
	cmdKubeSpawn.PersistentFlags().StringVarP(&k8sruntime, "container-runtime", "r", defaultRuntime, "Runtime to use for the spawned cluster (docker or rkt)")
	cmdKubeSpawn.PersistentFlags().StringVar(&rktBinDir, "rkt-bin-dir", "", "path to rkt binaries")
	cmdKubeSpawn.PersistentFlags().StringVar(&rktletBinDir, "rktlet-bin-dir", "", "path to rktlet binaries")
	cmdKubeSpawn.PersistentFlags().StringVar(&rktBinDir, "rkt-bin-dir", "", "path to rkt binaries")
	cmdKubeSpawn.PersistentFlags().StringVar(&rktletBinDir, "rktlet-bin-dir", "", "path to rktlet binaries")
	cmdKubeSpawn.PersistentFlags().StringVarP(&k8srelease, "kubernetes-version", "k", k8sStableVersion, "Kubernetes version to spawn, \"\" or \"dev\" for self-building upstream K8s.")
	cmdKubeSpawn.PersistentFlags().StringVarP(&kubeSpawnDir, "kube-spawn-dir", "d", kubeSpawnDirDefault, "path to kube-spawn asset directory")
}

func main() {
	if err := cmdKubeSpawn.Execute(); err != nil {
		log.Fatal(err)
	}
}

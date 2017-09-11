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
	"log"
	"os"
	"path"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

var (
	numNodes  int
	baseImage string

	cmdSetup = &cobra.Command{
		Use:   "setup",
		Short: "Set up nodes bringing up nspawn containers",
		Run:   runSetup,
	}
)

func init() {
	cmdKubeSpawn.AddCommand(cmdSetup)

	cmdSetup.Flags().IntVarP(&numNodes, "nodes", "n", 1, "number of nodes to spawn")
	cmdSetup.Flags().StringVarP(&baseImage, "image", "i", "coreos", "base image for nodes")
}

func checkK8sStableRelease(k8srel string) bool {
	v, err := semver.NewVersion(k8srel)
	if err != nil {
		// fallback to default version
		v, _ = semver.NewVersion(k8sStableVersion)
	}

	c, err := semver.NewConstraint(">=" + k8sStableVersion)
	if err != nil {
		log.Printf("Cannot get constraint for >= %s: %v", k8sStableVersion, err)
		return false
	}

	return c.Check(v)
}

func doCheckK8sStableRelease(k8srel string) {
	if !checkK8sStableRelease(k8srelease) {
		log.Printf("WARNING: K8s with version <%s is not compatible with kube-spawn.",
			k8sStableVersion)
		log.Printf("It's highly recommended to upgrade K8s to 1.7 or newer.")
		// Print a kind warning, and continue to run.
		// It should still allow kubeadm to run with Kubernetes <1.7,
		// to be able to allow end users to flexibly handle various cases.
	}
}

func runSetup(cmd *cobra.Command, args []string) {
	doSetup(numNodes, baseImage, kubeSpawnDir)
}

func doSetup(numNodes int, baseImage, kubeSpawnDir string) {
	if err := bootstrap.EnsureBridge(); err != nil {
		log.Fatalf("Error checking CNI bridge: %s", err)
	}

	if err := bootstrap.WriteNetConf(); err != nil {
		log.Fatalf("Error writing CNI configuration: %s", err)
	}

	log.Printf("Checking base image")
	if baseImage == "" {
		log.Fatal("No base image specified.")
	}
	if !bootstrap.MachineImageExists(baseImage) {
		log.Fatal("Base image not found.")
	}

	bootstrap.CreateSharedTmpdir()
	bootstrap.EnsureRequirements()

	doCheckK8sStableRelease(k8srelease)

	if !utils.IsK8sDev(k8srelease) {
		if err := bootstrap.DownloadK8sBins(k8srelease, path.Join(kubeSpawnDir, "k8s")); err != nil {
			log.Fatalf("Error downloading k8s files: %s", err)
		}
	}

	// NOTE: workaround for making kubelet work with port-forward.
	// Ideally we should solve the port-forward issue by either
	// creating general add-ons based on torcx, or creating our own
	// container image, or at least building socat statically on our own.
	ksExtraDir := path.Join(kubeSpawnDir, "extras")
	if err := os.MkdirAll(ksExtraDir, os.FileMode(0755)); err != nil {
		log.Fatalf("Unable to create directory %q: %v.", ksExtraDir, err)
	}
	if err := bootstrap.DownloadSocatBin(ksExtraDir); err != nil {
		log.Fatalf("Error downloading socat files: %s", err)
	}

	// estimate get pool size based on sum of virtual image sizes.
	var poolSize int64
	var err error
	if poolSize, err = bootstrap.GetPoolSize(baseImage, numNodes); err != nil {
		// fail hard in case of error, to avoid running unnecessary nodes
		log.Fatalf("cannot get pool size: %v", err)
	}

	var nodesToRun []nspawntool.Node

	for i := 0; i < numNodes; i++ {
		var node = nspawntool.Node{
			Name:       bootstrap.GetNodeName(i),
			K8sVersion: k8srelease,
			Runtime:    k8sruntime,
		}
		if !bootstrap.MachineImageExists(node.Name) {
			if err := bootstrap.NewNode(baseImage, node.Name); err != nil {
				log.Fatalf("Error cloning base image: %s", err)
			}
		}
		if bootstrap.IsNodeRunning(node.Name) {
			continue
		}
		nodesToRun = append(nodesToRun, node)
	}
	if len(nodesToRun) == 0 {
		log.Printf("No nodes to create. stop")
		os.Exit(1)
	}

	if err := bootstrap.EnlargeStoragePool(poolSize); err != nil {
		log.Printf("Warning: cannot enlarge storage pool: %v", err)
	}

	for _, node := range nodesToRun {
		if err := node.Run(kubeSpawnDir); err != nil {
			log.Fatalf("Error running node: %s", err)
		}

		if err := node.Bootstrap(); err != nil {
			log.Fatalf("Error running bootstrap script: %s", err)
		}

		log.Printf("Success! %s started.", node.Name)
	}

	log.Printf("All nodes are running. Use machinectl to login/stop/etc.")
}

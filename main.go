package main

import (
	"context"
	"flag"
	"path/filepath"

	"gh.io/akj/devops-skills-assessment/utils"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var logLevel string
var podContains string
var wait bool

func init() {
	flag.StringVar(&logLevel, "log-level", "info", "Logging level (panic, fatal, error, warning, info, debug, trace)")
	flag.StringVar(&podContains, "pod-contains", "database", "Restart pods that contain this string")
	flag.BoolVar(&wait, "wait", false, "Wait for the old set of pods to be removed")
	flag.Parse()

	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(lvl)
}

func main() {
	ctx := context.Background()

	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		log.Fatalf("could not build config from kubeconfig: %s", err)
	}

	client, err := utils.CreateClient(config)
	if err != nil {
		log.Fatal(err)
	}
	if restartCount, err := utils.Restart(ctx, client, podContains, wait); err != nil {
		log.Fatal(err)
	} else {
		log.Infof("restarted %d pod(s) that contain `%s` in the name", restartCount, podContains)
	}
}

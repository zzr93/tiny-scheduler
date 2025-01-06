package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/zzr93/tiny-scheduler/pkg/kube"
	"github.com/zzr93/tiny-scheduler/pkg/scheduler"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := kube.InitKubeClient(); err != nil {
		glog.Fatal(err)
	}

	scheduler.InitScheduler(&scheduler.Config{
		Name: "tiny-scheduler",
	})

	select {}
}

package main

import (
	"github.com/reddec/gopack/pack"
	"log"
	"flag"
	"path"
)

func main() {
	dir := flag.String("d", ".", "Directory with package.json")
	create := flag.Bool("c", false, "Create package instead of build")
	service := flag.Bool("s", false, "Create a service stub (for -c)")
	output := flag.String("o", ".", "Output directory")
	flag.Parse()

	if *create && flag.NArg() == 0 {
		panic("Requires package name")
	}

	if *create {
		if *service {
			pack.SaveNewService(*dir, flag.Arg(0))
		}else {
			pack.SaveNewApp(*dir, flag.Arg(0))
		}
		return
	}



	log.Println("Building")
	p, err := pack.ReadPackage(path.Join(*dir, "package.json"))
	if err != nil {
		panic(err)
	}
	err = p.Make(*output)
	if err != nil {
		panic(err)
	}
	log.Println("Done")
}


package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	//	"github.com/mitchellh/go-ps"
)

func main() {
	signalReceived := make(chan bool)
	setupInterruptCapture(signalReceived)
	pid := os.Getpid()
	err := syscall.Setpgid(pid, pid) // set own pid as gpid to also set it for child processes
	if err != nil {
		panic(fmt.Sprintf("Failed to setpgid]: %d - %s\n", pid, err))
	}

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		fmt.Printf("Starting main program with PID %d\n", pid)
		// main program, out CLI
		execute("spawner", pid)
		execute("spawner", pid)
		fmt.Printf("\n\n All set up! Now do a 'kill %d' to terminate the main process.\n Then look for child processes with 'ps ax | grep longrunning'.\n\n", pid)
		<-signalReceived

		cleanProcesses()
	} else {
		switch argsWithoutProg[0] {
		case "spawner":
			// this is the yarn command
			// note that you can not do any process management here, as we have no control over this process usually
			fmt.Printf("Starting spawner with PID %d\n", pid)
			// NOTE that we are not allowed to modify the GPID!
			execute("longrunning", 0)
		case "longrunning":
			// this is the node process, started by the yarn command
			fmt.Printf("Starting long running with PID %d\n", pid)
			<-signalReceived
		}
	}
	fmt.Printf("PID %d is exiting\n", pid)
}

func execute(arg string, pid int) *exec.Cmd {
	executable, err := os.Executable()
	if err != nil {
		panic(fmt.Sprintf("could not get path to executable: %s", err))
	}
	cmd := exec.Command(executable, arg)
	if pid != 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pid, Setsid: true}
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Running: %v, args: %v, os.Args: %v\n", cmd, cmd.Args, os.Args)
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	return cmd
}

func setupInterruptCapture(receivedCh chan bool) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received interrupt signal, cleaning up...")
		// do cleanup here
		select {
		case receivedCh <- true:
			fmt.Println("sent signal to main")
		default:
			fmt.Println("channel is not ready")
		}
	}()
}

func cleanProcesses() {
	pid := os.Getpid()
	// send signal to process (negative process id)
	gpid := -pid
	fmt.Printf("Sending sigint to process %d\n", gpid)
	if e := syscall.Kill(gpid, syscall.SIGINT); e != nil {
		panic(e)
	}
	fmt.Println("Terminated.")
}

// does not work, because the spawner end themselves and longrunning become child of PID 1
/*
func CleanProcessesByParent() {
	thisPid := os.Getpid()
	processes, err := ps.Processes()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got %d processes", len(processes))
	for _, p := range processes {
		if p.PPid() == thisPid {
			fmt.Printf("Found child process: %v", p)
			proc, err := os.FindProcess(p.Pid())
			if err != nil {
				panic(err)
			}
			fmt.Printf("will kill process with id %d, which is %v\n", p.Pid(), proc)

			err = proc.Signal(syscall.SIGINT)
			if err != nil {
				panic(err)
			}
			fmt.Println("Terminated.")
		}
	}
}
*/

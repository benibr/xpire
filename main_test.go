package main

import (
    "os/exec"
    "testing"
)

func TestMain(t *testing.T) {
    tests := []struct {
        args    []string
        want    string
    }{
        {[]string{"--path", "./", "--prune"}, "pruning all expired snapshots in './'"},
        //{[]string{"--path", "./", "-s", "foobar"}, "parsing time \"foobar\" as \"2006-01-02 15:04:05\": cannot parse \"foobar\" as \"2006\" \nexit status 5\n"},
        {[]string{"--path", "./", "-s", "2022-06-30 15:38:13"}, "setting expiration date on snapshot './' to 2022-06-30 15:38:13"},
    }

    for _, tt := range tests {
        t.Run("Testing with args "+tt.args[0], func(t *testing.T) {
            cmd := exec.Command("go", "run", "main.go")
            cmd.Args = append(cmd.Args, tt.args...)

            output, err := cmd.CombinedOutput()
            if got := string(output); got != tt.want {
                t.Errorf("\n got output: '%v'\nwant output: '%v'", got, tt.want)
            }
            if err != nil {
                t.Fatalf("Failed to run command: %v", err)
            }

        })
    }
}

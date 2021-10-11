package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	api "github.com/spirifoxy/teleworker/internal/api/v1"
	"github.com/spirifoxy/teleworker/server/internal/auth"
	"github.com/spirifoxy/teleworker/server/internal/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// assertExpectedAndActual is a helper function to allow the step function to call
// assertion functions where you want to compare an expected and an actual value.
func assertExpectedAndActual(a expectedAndActualAssertion, expected, actual interface{}, msgAndArgs ...interface{}) error {
	var t asserter
	a(&t, expected, actual, msgAndArgs...)
	return t.err
}

type expectedAndActualAssertion func(t assert.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool

// asserter is used to be able to retrieve the error reported by the called assertion
type asserter struct {
	err error
}

// Errorf is used by the called assertion to report an error
func (a *asserter) Errorf(format string, args ...interface{}) {
	a.err = fmt.Errorf(format, args...)
}

const bufSize = 1024 * 1024

var listener *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return listener.Dial()
}

func initSuite() {
	UsernameFromCtx = func(ctx context.Context) (*auth.User, bool) {
		return &auth.User{Name: "test_client"}, true
	}

	listener = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	twServer := &TWServer{
		store: storage.NewMemStorage(),
	}

	api.RegisterTeleWorkerServer(grpcServer, twServer)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()

}

// TestMain allows us to run e2e tests using go test command,
// which might be useful for debugging or running all the tests
// with only one command
func TestMain(m *testing.M) {
	opts := godog.Options{
		Format:    "pretty",
		Paths:     []string{"test/features"},
		Randomize: time.Now().UTC().UnixNano(), // randomize scenario execution order
	}

	status := godog.TestSuite{
		Name:                 "tw_tests",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &opts,
	}.Run()

	os.Exit(status)
}

func InitializeTestSuite(sc *godog.TestSuiteContext) {
	sc.BeforeSuite(initSuite)
}

type State struct {
	ctx       context.Context
	command   string
	arguments []string

	lastError error
	subject   interface{}
}

var scenarioState *State

type clientFeature struct {
	con    *grpc.ClientConn
	client api.TeleWorkerClient
}

var f *clientFeature

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		con, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
		if err != nil {
			return nil, err
		}

		scenarioState = &State{
			ctx: ctx,
		}
		f = &clientFeature{
			con:    con,
			client: api.NewTeleWorkerClient(con),
		}
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		f.con.Close()
		scenarioState = nil

		return ctx, nil
	})

	ctx.Step(`^the response is success$`, theResponseIsSuccess)
	ctx.Step(`^the response is error$`, theResponseIsError)
	ctx.Step(`^the job was created$`, theJobWasCreated)
	ctx.Step(`^I wait for a second$`, iWaitForASecond)

	// start
	ctx.Step(`^I pass my command (.*)$`, iPassMyCommand)
	ctx.Step(`^I pass command argument (.*)$`, iPassCommandArgument)
	ctx.Step(`^I try to create new job$`, iTryToCreateNewJob)
	ctx.Step(`^I get the job uuid$`, iGetTheJobUuid)

	// stop
	ctx.Step(`^I try to stop the job$`, iTryToStopTheJob)
	ctx.Step(`^I try to stop some random job$`, iTryToStopSomeRandomJob)

	// status
	ctx.Step(`^I see the job is finished$`, iSeeTheJobIsFinished)
	ctx.Step(`^I see the job is still running$`, iSeeTheJobIsStillRunning)
	ctx.Step(`^I try to get status of the job$`, iTryToGetStatusOfTheJob)
}

func theResponseIsSuccess() error {
	return assertExpectedAndActual(
		assert.IsType, nil, scenarioState.lastError,
		"expected response to be successfull, but error happened: %v", scenarioState.lastError,
	)
}

func theResponseIsError() error {
	if scenarioState.lastError == nil {
		return fmt.Errorf("expected error to happen, but it did not")
	}
	return nil
}

func theJobWasCreated() error {
	iTryToCreateNewJob()
	return iGetTheJobUuid()
}

func iWaitForASecond() error {
	// required in order give the job time to finish
	time.Sleep(time.Second)
	return nil
}

/********************/
// start steps
/********************/
func iPassMyCommand(command string) error {
	scenarioState.command = command
	return nil
}

func iPassCommandArgument(arg string) error {
	scenarioState.arguments = append(scenarioState.arguments, arg)
	return nil
}

func iTryToCreateNewJob() error {
	scenarioState.subject, scenarioState.lastError = f.client.Start(scenarioState.ctx, &api.StartRequest{
		Command: scenarioState.command,
		Args:    scenarioState.arguments,
	})
	return nil
}

func iGetTheJobUuid() error {
	resp, ok := scenarioState.subject.(*api.StartResponse)
	if !ok {
		return fmt.Errorf("expected to receive StartResponse, but failed")
	}
	uuid := resp.GetJobId()
	return assertExpectedAndActual(
		assert.Equal, 36, len(uuid),
		fmt.Sprintf("expected to receive job uuid, but received: %s", uuid),
	)
}

/********************/
// stop steps
/********************/
func iTryToStopTheJob() error {
	resp := scenarioState.subject.(*api.StartResponse)
	uuid := resp.GetJobId()
	_, scenarioState.lastError = f.client.Stop(scenarioState.ctx, &api.StopRequest{
		JobId: uuid,
	})
	return nil
}

func iTryToStopSomeRandomJob() error {
	uuid := "42"
	_, scenarioState.lastError = f.client.Stop(scenarioState.ctx, &api.StopRequest{
		JobId: uuid,
	})
	return nil
}

/********************/
// status steps
/********************/
func iTryToGetStatusOfTheJob() error {
	resp := scenarioState.subject.(*api.StartResponse)
	uuid := resp.GetJobId()
	scenarioState.subject, scenarioState.lastError = f.client.Status(scenarioState.ctx, &api.StatusRequest{
		JobId: uuid,
	})
	return nil
}

func iSeeTheJobIsFinished() error {
	resp, ok := scenarioState.subject.(*api.StatusResponse)
	if !ok {
		return fmt.Errorf("expected to receive StatusResponse, but failed")
	}

	return assertExpectedAndActual(
		assert.Equal, api.JobStatus_FINISHED.String(), resp.Status.String(),
		fmt.Sprintf("expected the job to be finished, but received: %s", resp.Status.String()),
	)
}

func iSeeTheJobIsStillRunning() error {
	resp, ok := scenarioState.subject.(*api.StatusResponse)
	if !ok {
		return fmt.Errorf("expected to receive StatusResponse, but failed")
	}

	return assertExpectedAndActual(
		assert.Equal, api.JobStatus_ALIVE.String(), resp.Status.String(),
		fmt.Sprintf("expected the job to be running, but received: %s", resp.Status.String()),
	)
}

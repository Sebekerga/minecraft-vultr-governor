package create_server

import (
	"context"
	"fmt"
	"log"
	"os"
	mcvultrgov "sebekerga/vultr_minecraft_governor"
	routines "sebekerga/vultr_minecraft_governor/routines"
	"strconv"
	"time"

	"github.com/melbahja/goph"
	"github.com/vultr/govultr/v3"
)

type CreatingServerContext struct {
	VCtx        context.Context
	VultrClient *govultr.Client

	SSHClient *goph.Client

	TargetInstanceLabel  string
	TargetInstanceRegion string
	TargetInstancePlan   string
	TargetInstanceOSID   int
	TargetScriptID       string
	TargetBlockLabel     string

	createdInstanceID string
	createdInstanceIP string
	targetBlockID     string

	attemptCounter routines.AttemptCounter
}

func InitContext(vultrContext context.Context, vultrClient *govultr.Client) CreatingServerContext {
	instance_os_id, err := strconv.Atoi(os.Getenv(mcvultrgov.TARGET_INSTANCE_OS_ID_KEY))
	if err != nil {
		panic(err)
	}

	return CreatingServerContext{
		VCtx:        vultrContext,
		VultrClient: vultrClient,

		TargetInstanceLabel:  os.Getenv(mcvultrgov.TARGET_INSTANCE_LABEL_KEY),
		TargetInstanceRegion: os.Getenv(mcvultrgov.TARGET_INSTANCE_REGION_KEY),
		TargetInstancePlan:   os.Getenv(mcvultrgov.TARGET_INSTANCE_PLAN_KEY),
		TargetInstanceOSID:   instance_os_id,
		TargetScriptID:       os.Getenv(mcvultrgov.TARGET_SCRIPT_ID_KEY),
		TargetBlockLabel:     os.Getenv(mcvultrgov.TARGET_BLOCK_LABEL_KEY),

		attemptCounter: routines.NewAttemptCounter(),
	}
}

type Ctx = CreatingServerContext
type F = routines.RoutineFunc[Ctx]

func CreatingServerEntry(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	return _CheckIfInstanceExists, nil
}

func _CheckIfInstanceExists(ctx *Ctx, ph routines.PrintHandler) (F, error) {

	instances, _, _, err := ctx.VultrClient.Instance.List(ctx.VCtx, &govultr.ListOptions{})
	if err != nil {
		ph(routines.ERROR, "Error occurred while fetching instances")
		return nil, err
	}

	for _, instance := range instances {
		if instance.Label == ctx.TargetInstanceLabel {
			ctx.createdInstanceID = instance.ID
			ctx.createdInstanceIP = instance.MainIP
			ph(routines.INFO, fmt.Sprintf("Found existing instance, ID: %s", ctx.createdInstanceID))
			return _FindBlockStorage, nil
		}
	}

	ph(routines.INFO, "No existing instance found")
	return _CreateInstance, nil
}

func _CreateInstance(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Creating new instance")
	instance, _, err := ctx.VultrClient.Instance.Create(ctx.VCtx, &govultr.InstanceCreateReq{
		Label:    ctx.TargetInstanceLabel,
		Region:   ctx.TargetInstanceRegion,
		Plan:     ctx.TargetInstancePlan,
		OsID:     ctx.TargetInstanceOSID,
		ScriptID: ctx.TargetScriptID,
	})
	if err != nil {
		ph(routines.ERROR, "Error occurred while creating instance")
		return nil, err
	}
	ctx.createdInstanceID = instance.ID
	ctx.createdInstanceIP = instance.MainIP
	ph(routines.INFO, fmt.Sprintf("Instance created, ID: %s", ctx.createdInstanceID))

	return _WaitForInstanceToBeCreated, nil
}

func _WaitForInstanceToBeCreated(ctx *Ctx, ph routines.PrintHandler) (F, error) {

	const MAX_ATTEMPTS = 10
	const ACTION_ID = "wait_for_instance"

	if ctx.attemptCounter.Get(ACTION_ID) <= 0 {
		ph(routines.INFO, "Waiting for instance to be created")
	}

	instance, _, err := ctx.VultrClient.Instance.Get(ctx.VCtx, ctx.createdInstanceID)
	if err != nil {
		ph(routines.ERROR, "Error occurred while fetching instance")
		return nil, err
	}

	if instance.Status != "active" || instance.PowerStatus != "running" || instance.ServerStatus != "ok" {
		ctx.attemptCounter.Increment(ACTION_ID)
		ph(routines.INFO, fmt.Sprintf("Attempt %d: Instance not active yet", ctx.attemptCounter.Get(ACTION_ID)))
		if ctx.attemptCounter.Get(ACTION_ID) >= MAX_ATTEMPTS {
			ph(routines.ERROR, "Max attempts reached")
			return nil, fmt.Errorf("max attempts reached")
		}
		time.Sleep(10 * time.Second)
		return _WaitForInstanceToBeCreated, nil
	}

	ph(routines.INFO, "Instance active")
	return _FindBlockStorage, nil
}

func _FindBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Checking if block storage already created")
	blocks, _, _, err := ctx.VultrClient.BlockStorage.List(ctx.VCtx, &govultr.ListOptions{})
	if err != nil {
		ph(routines.ERROR, "Error occurred while fetching block storage")
		return nil, err
	}

	for _, block := range blocks {
		if block.Label == ctx.TargetBlockLabel {
			ctx.targetBlockID = block.ID
			ph(routines.INFO, fmt.Sprintf("Found existing block storage, ID: %s", ctx.targetBlockID))
			return _AwaitServerSSH, nil
		}
	}

	return _CreateBlockStorage, nil
}

func _CreateBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Creating new block storage")
	block, _, err := ctx.VultrClient.BlockStorage.Create(ctx.VCtx, &govultr.BlockStorageCreate{
		Region: ctx.TargetInstanceRegion,
		Label:  ctx.TargetBlockLabel,
		SizeGB: 10,
	})
	if err != nil {
		ph(routines.ERROR, "Error occurred while creating block storage")
		return nil, err
	}
	ctx.targetBlockID = block.ID
	ph(routines.INFO, fmt.Sprintf("Block storage created, ID: %s", ctx.targetBlockID))

	return _AwaitServerSSH, nil
}

func _AwaitServerSSH(ctx *Ctx, ph routines.PrintHandler) (F, error) {

	const MAX_ATTEMPTS = 5
	const ACTION_ID = "await_server_ssh"

	if ctx.attemptCounter.Get(ACTION_ID) <= 0 {
		ph(routines.INFO, fmt.Sprintf("Waiting for server SSH, IP: %s", ctx.createdInstanceIP))
	}

	keyPath := os.Getenv(mcvultrgov.INSTANCE_SSH_KEY_PATH_KEY)
	auth, err := goph.Key(keyPath, "")
	if err != nil {
		ph(routines.ERROR, "Unable to get SSH key")
		return nil, err
	}
	ph(routines.INFO, "Loaded key")

	// I cant know host key before server is booted
	sshClient, err := goph.NewUnknown("root", ctx.createdInstanceIP, auth)
	if err != nil {
		log.Printf("Error: %s", err)
		ctx.attemptCounter.Increment(ACTION_ID)
		ph(routines.INFO, fmt.Sprintf("Attempt %d: Server not booted yet", ctx.attemptCounter.Get(ACTION_ID)))
		if ctx.attemptCounter.Get(ACTION_ID) >= MAX_ATTEMPTS {
			ph(routines.ERROR, "Max attempts reached")
			return nil, err
		}
		time.Sleep(10 * time.Second)
		return _AwaitServerSSH, nil
	}

	ctx.SSHClient = sshClient

	ph(routines.INFO, "SSH connection established")
	return _AttachBlockStorage, nil
}

func _AttachBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	ph(routines.INFO, "Attaching block storage")
	err := ctx.VultrClient.BlockStorage.Attach(ctx.VCtx, ctx.targetBlockID, &govultr.BlockStorageAttach{
		InstanceID: ctx.createdInstanceID,
		Live:       govultr.BoolToBoolPtr(true),
	})
	if err != nil {
		ph(routines.ERROR, "Error occurred while attaching block storage")
		return nil, err
	}
	ph(routines.INFO, "Block storage attached")

	return _MountingBlockStorage, nil
}

func _MountingBlockStorage(ctx *Ctx, ph routines.PrintHandler) (F, error) {
	_, err := ctx.SSHClient.Run("mkdir /mnt/minecraft")
	if err != nil {
		ph(routines.ERROR, "Error occurred while creating directory")
		return nil, err
	}
	ph(routines.INFO, "Directory created")

	_, err = ctx.SSHClient.Run("mount /dev/sda /mnt/minecraft")
	if err != nil {
		ph(routines.ERROR, "Error occurred while mounting block storage")
		return nil, err
	}
	ph(routines.INFO, "Block storage mounted")

	return nil, nil
}

// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: clusterinst.proto

package gencmd

import (
	"context"
	fmt "fmt"
	_ "github.com/edgexr/edge-cloud-platform/api/distributed_match_engine"
	distributed_match_engine "github.com/edgexr/edge-cloud-platform/api/distributed_match_engine"
	edgeproto "github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/cli"
	_ "github.com/edgexr/edge-cloud-platform/tools/protogen"
	_ "github.com/gogo/googleapis/google/api"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
	"io"
	math "math"
	"strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Auto-generated code: DO NOT EDIT
func ClusterInstHideTags(in *edgeproto.ClusterInst) {
	if cli.HideTags == "" {
		return
	}
	tags := make(map[string]struct{})
	for _, tag := range strings.Split(cli.HideTags, ",") {
		tags[tag] = struct{}{}
	}
	if _, found := tags["nocmp"]; found {
		in.Errors = nil
	}
	if _, found := tags["nocmp"]; found {
		in.AllocatedIp = ""
	}
	for i1 := 0; i1 < len(in.Resources.Vms); i1++ {
		for i2 := 0; i2 < len(in.Resources.Vms[i1].Ipaddresses); i2++ {
		}
		for i2 := 0; i2 < len(in.Resources.Vms[i1].Containers); i2++ {
		}
	}
	if _, found := tags["timestamp"]; found {
		in.CreatedAt = distributed_match_engine.Timestamp{}
	}
	if _, found := tags["timestamp"]; found {
		in.UpdatedAt = distributed_match_engine.Timestamp{}
	}
	if _, found := tags["timestamp"]; found {
		in.ReservationEndedAt = distributed_match_engine.Timestamp{}
	}
	if _, found := tags["nocmp"]; found {
		in.ObjId = ""
	}
	if _, found := tags["nocmp"]; found {
		in.CompatibilityVersion = 0
	}
	if _, found := tags["nocmp"]; found {
		in.DbModelId = 0
	}
	for i0 := 0; i0 < len(in.NodePools); i0++ {
	}
}

func ClusterInstInfoHideTags(in *edgeproto.ClusterInstInfo) {
	if cli.HideTags == "" {
		return
	}
	tags := make(map[string]struct{})
	for _, tag := range strings.Split(cli.HideTags, ",") {
		tags[tag] = struct{}{}
	}
	if _, found := tags["nocmp"]; found {
		in.NotifyId = 0
	}
	for i1 := 0; i1 < len(in.Resources.Vms); i1++ {
		for i2 := 0; i2 < len(in.Resources.Vms[i1].Ipaddresses); i2++ {
		}
		for i2 := 0; i2 < len(in.Resources.Vms[i1].Containers); i2++ {
		}
	}
}

var ClusterInstApiCmd edgeproto.ClusterInstApiClient

var CreateClusterInstCmd = &cli.Command{
	Use:          "CreateClusterInst",
	RequiredArgs: strings.Join(ClusterInstRequiredArgs, " "),
	OptionalArgs: strings.Join(ClusterInstOptionalArgs, " "),
	AliasArgs:    strings.Join(ClusterInstAliasArgs, " "),
	SpecialArgs:  &ClusterInstSpecialArgs,
	Comments:     ClusterInstComments,
	ReqData:      &edgeproto.ClusterInst{},
	ReplyData:    &edgeproto.Result{},
	Run:          runCreateClusterInst,
}

func runCreateClusterInst(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.ClusterInst)
	_, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	return CreateClusterInst(c, obj)
}

func CreateClusterInst(c *cli.Command, in *edgeproto.ClusterInst) error {
	if ClusterInstApiCmd == nil {
		return fmt.Errorf("ClusterInstApi client not initialized")
	}
	ctx := context.Background()
	stream, err := ClusterInstApiCmd.CreateClusterInst(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("CreateClusterInst failed: %s", errstr)
	}

	objs := make([]*edgeproto.Result, 0)
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errstr := err.Error()
			st, ok := status.FromError(err)
			if ok {
				errstr = st.Message()
			}
			return fmt.Errorf("CreateClusterInst recv failed: %s", errstr)
		}
		if cli.OutputStream {
			c.WriteOutput(c.CobraCmd.OutOrStdout(), obj, cli.OutputFormat)
			continue
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), objs, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func CreateClusterInsts(c *cli.Command, data []edgeproto.ClusterInst, err *error) {
	if *err != nil {
		return
	}
	for ii, _ := range data {
		fmt.Printf("CreateClusterInst %v\n", data[ii])
		myerr := CreateClusterInst(c, &data[ii])
		if myerr != nil {
			*err = myerr
			break
		}
	}
}

var DeleteClusterInstCmd = &cli.Command{
	Use:          "DeleteClusterInst",
	RequiredArgs: strings.Join(ClusterInstRequiredArgs, " "),
	OptionalArgs: strings.Join(ClusterInstOptionalArgs, " "),
	AliasArgs:    strings.Join(ClusterInstAliasArgs, " "),
	SpecialArgs:  &ClusterInstSpecialArgs,
	Comments:     ClusterInstComments,
	ReqData:      &edgeproto.ClusterInst{},
	ReplyData:    &edgeproto.Result{},
	Run:          runDeleteClusterInst,
}

func runDeleteClusterInst(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.ClusterInst)
	_, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	return DeleteClusterInst(c, obj)
}

func DeleteClusterInst(c *cli.Command, in *edgeproto.ClusterInst) error {
	if ClusterInstApiCmd == nil {
		return fmt.Errorf("ClusterInstApi client not initialized")
	}
	ctx := context.Background()
	stream, err := ClusterInstApiCmd.DeleteClusterInst(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("DeleteClusterInst failed: %s", errstr)
	}

	objs := make([]*edgeproto.Result, 0)
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errstr := err.Error()
			st, ok := status.FromError(err)
			if ok {
				errstr = st.Message()
			}
			return fmt.Errorf("DeleteClusterInst recv failed: %s", errstr)
		}
		if cli.OutputStream {
			c.WriteOutput(c.CobraCmd.OutOrStdout(), obj, cli.OutputFormat)
			continue
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), objs, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func DeleteClusterInsts(c *cli.Command, data []edgeproto.ClusterInst, err *error) {
	if *err != nil {
		return
	}
	for ii, _ := range data {
		fmt.Printf("DeleteClusterInst %v\n", data[ii])
		myerr := DeleteClusterInst(c, &data[ii])
		if myerr != nil {
			*err = myerr
			break
		}
	}
}

var UpdateClusterInstCmd = &cli.Command{
	Use:          "UpdateClusterInst",
	RequiredArgs: strings.Join(UpdateClusterInstRequiredArgs, " "),
	OptionalArgs: strings.Join(UpdateClusterInstOptionalArgs, " "),
	AliasArgs:    strings.Join(ClusterInstAliasArgs, " "),
	SpecialArgs:  &ClusterInstSpecialArgs,
	Comments:     ClusterInstComments,
	ReqData:      &edgeproto.ClusterInst{},
	ReplyData:    &edgeproto.Result{},
	Run:          runUpdateClusterInst,
}

func runUpdateClusterInst(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.ClusterInst)
	jsonMap, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	obj.Fields = cli.GetSpecifiedFields(jsonMap, c.ReqData)
	return UpdateClusterInst(c, obj)
}

func UpdateClusterInst(c *cli.Command, in *edgeproto.ClusterInst) error {
	if ClusterInstApiCmd == nil {
		return fmt.Errorf("ClusterInstApi client not initialized")
	}
	ctx := context.Background()
	stream, err := ClusterInstApiCmd.UpdateClusterInst(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("UpdateClusterInst failed: %s", errstr)
	}

	objs := make([]*edgeproto.Result, 0)
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errstr := err.Error()
			st, ok := status.FromError(err)
			if ok {
				errstr = st.Message()
			}
			return fmt.Errorf("UpdateClusterInst recv failed: %s", errstr)
		}
		if cli.OutputStream {
			c.WriteOutput(c.CobraCmd.OutOrStdout(), obj, cli.OutputFormat)
			continue
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), objs, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func UpdateClusterInsts(c *cli.Command, data []edgeproto.ClusterInst, err *error) {
	if *err != nil {
		return
	}
	for ii, _ := range data {
		fmt.Printf("UpdateClusterInst %v\n", data[ii])
		myerr := UpdateClusterInst(c, &data[ii])
		if myerr != nil {
			*err = myerr
			break
		}
	}
}

var ShowClusterInstCmd = &cli.Command{
	Use:          "ShowClusterInst",
	OptionalArgs: strings.Join(append(ClusterInstRequiredArgs, ClusterInstOptionalArgs...), " "),
	AliasArgs:    strings.Join(ClusterInstAliasArgs, " "),
	SpecialArgs:  &ClusterInstSpecialArgs,
	Comments:     ClusterInstComments,
	ReqData:      &edgeproto.ClusterInst{},
	ReplyData:    &edgeproto.ClusterInst{},
	Run:          runShowClusterInst,
}

func runShowClusterInst(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.ClusterInst)
	_, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	return ShowClusterInst(c, obj)
}

func ShowClusterInst(c *cli.Command, in *edgeproto.ClusterInst) error {
	if ClusterInstApiCmd == nil {
		return fmt.Errorf("ClusterInstApi client not initialized")
	}
	ctx := context.Background()
	stream, err := ClusterInstApiCmd.ShowClusterInst(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("ShowClusterInst failed: %s", errstr)
	}

	objs := make([]*edgeproto.ClusterInst, 0)
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errstr := err.Error()
			st, ok := status.FromError(err)
			if ok {
				errstr = st.Message()
			}
			return fmt.Errorf("ShowClusterInst recv failed: %s", errstr)
		}
		ClusterInstHideTags(obj)
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), objs, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func ShowClusterInsts(c *cli.Command, data []edgeproto.ClusterInst, err *error) {
	if *err != nil {
		return
	}
	for ii, _ := range data {
		fmt.Printf("ShowClusterInst %v\n", data[ii])
		myerr := ShowClusterInst(c, &data[ii])
		if myerr != nil {
			*err = myerr
			break
		}
	}
}

var DeleteIdleReservableClusterInstsCmd = &cli.Command{
	Use:          "DeleteIdleReservableClusterInsts",
	RequiredArgs: strings.Join(IdleReservableClusterInstsRequiredArgs, " "),
	OptionalArgs: strings.Join(IdleReservableClusterInstsOptionalArgs, " "),
	AliasArgs:    strings.Join(IdleReservableClusterInstsAliasArgs, " "),
	SpecialArgs:  &IdleReservableClusterInstsSpecialArgs,
	Comments:     IdleReservableClusterInstsComments,
	ReqData:      &edgeproto.IdleReservableClusterInsts{},
	ReplyData:    &edgeproto.Result{},
	Run:          runDeleteIdleReservableClusterInsts,
}

func runDeleteIdleReservableClusterInsts(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.IdleReservableClusterInsts)
	_, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	return DeleteIdleReservableClusterInsts(c, obj)
}

func DeleteIdleReservableClusterInsts(c *cli.Command, in *edgeproto.IdleReservableClusterInsts) error {
	if ClusterInstApiCmd == nil {
		return fmt.Errorf("ClusterInstApi client not initialized")
	}
	ctx := context.Background()
	obj, err := ClusterInstApiCmd.DeleteIdleReservableClusterInsts(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("DeleteIdleReservableClusterInsts failed: %s", errstr)
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), obj, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func DeleteIdleReservableClusterInstsBatch(c *cli.Command, data *edgeproto.IdleReservableClusterInsts, err *error) {
	if *err != nil || data == nil {
		return
	}
	fmt.Printf("DeleteIdleReservableClusterInsts %v\n", data)
	myerr := DeleteIdleReservableClusterInsts(c, data)
	if myerr != nil {
		*err = myerr
	}
}

var ShowClusterResourceUsageCmd = &cli.Command{
	Use:          "ShowClusterResourceUsage",
	RequiredArgs: strings.Join(ShowClusterResourceUsageRequiredArgs, " "),
	OptionalArgs: strings.Join(ShowClusterResourceUsageOptionalArgs, " "),
	AliasArgs:    strings.Join(ClusterInstAliasArgs, " "),
	SpecialArgs:  &ClusterInstSpecialArgs,
	Comments:     ClusterInstComments,
	ReqData:      &edgeproto.ClusterInst{},
	ReplyData:    &edgeproto.ClusterResourceUsage{},
	Run:          runShowClusterResourceUsage,
}

func runShowClusterResourceUsage(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.ClusterInst)
	_, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	return ShowClusterResourceUsage(c, obj)
}

func ShowClusterResourceUsage(c *cli.Command, in *edgeproto.ClusterInst) error {
	if ClusterInstApiCmd == nil {
		return fmt.Errorf("ClusterInstApi client not initialized")
	}
	ctx := context.Background()
	stream, err := ClusterInstApiCmd.ShowClusterResourceUsage(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("ShowClusterResourceUsage failed: %s", errstr)
	}

	objs := make([]*edgeproto.ClusterResourceUsage, 0)
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errstr := err.Error()
			st, ok := status.FromError(err)
			if ok {
				errstr = st.Message()
			}
			return fmt.Errorf("ShowClusterResourceUsage recv failed: %s", errstr)
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), objs, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func ShowClusterResourceUsages(c *cli.Command, data []edgeproto.ClusterInst, err *error) {
	if *err != nil {
		return
	}
	for ii, _ := range data {
		fmt.Printf("ShowClusterResourceUsage %v\n", data[ii])
		myerr := ShowClusterResourceUsage(c, &data[ii])
		if myerr != nil {
			*err = myerr
			break
		}
	}
}

var ClusterInstApiCmds = []*cobra.Command{
	CreateClusterInstCmd.GenCmd(),
	DeleteClusterInstCmd.GenCmd(),
	UpdateClusterInstCmd.GenCmd(),
	ShowClusterInstCmd.GenCmd(),
	DeleteIdleReservableClusterInstsCmd.GenCmd(),
	ShowClusterResourceUsageCmd.GenCmd(),
}

var ClusterInstInfoApiCmd edgeproto.ClusterInstInfoApiClient

var ShowClusterInstInfoCmd = &cli.Command{
	Use:          "ShowClusterInstInfo",
	OptionalArgs: strings.Join(append(ClusterInstInfoRequiredArgs, ClusterInstInfoOptionalArgs...), " "),
	AliasArgs:    strings.Join(ClusterInstInfoAliasArgs, " "),
	SpecialArgs:  &ClusterInstInfoSpecialArgs,
	Comments:     ClusterInstInfoComments,
	ReqData:      &edgeproto.ClusterInstInfo{},
	ReplyData:    &edgeproto.ClusterInstInfo{},
	Run:          runShowClusterInstInfo,
}

func runShowClusterInstInfo(c *cli.Command, args []string) error {
	if cli.SilenceUsage {
		c.CobraCmd.SilenceUsage = true
	}
	obj := c.ReqData.(*edgeproto.ClusterInstInfo)
	_, err := c.ParseInput(args)
	if err != nil {
		return err
	}
	return ShowClusterInstInfo(c, obj)
}

func ShowClusterInstInfo(c *cli.Command, in *edgeproto.ClusterInstInfo) error {
	if ClusterInstInfoApiCmd == nil {
		return fmt.Errorf("ClusterInstInfoApi client not initialized")
	}
	ctx := context.Background()
	stream, err := ClusterInstInfoApiCmd.ShowClusterInstInfo(ctx, in)
	if err != nil {
		errstr := err.Error()
		st, ok := status.FromError(err)
		if ok {
			errstr = st.Message()
		}
		return fmt.Errorf("ShowClusterInstInfo failed: %s", errstr)
	}

	objs := make([]*edgeproto.ClusterInstInfo, 0)
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errstr := err.Error()
			st, ok := status.FromError(err)
			if ok {
				errstr = st.Message()
			}
			return fmt.Errorf("ShowClusterInstInfo recv failed: %s", errstr)
		}
		ClusterInstInfoHideTags(obj)
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}
	c.WriteOutput(c.CobraCmd.OutOrStdout(), objs, cli.OutputFormat)
	return nil
}

// this supports "Create" and "Delete" commands on ApplicationData
func ShowClusterInstInfos(c *cli.Command, data []edgeproto.ClusterInstInfo, err *error) {
	if *err != nil {
		return
	}
	for ii, _ := range data {
		fmt.Printf("ShowClusterInstInfo %v\n", data[ii])
		myerr := ShowClusterInstInfo(c, &data[ii])
		if myerr != nil {
			*err = myerr
			break
		}
	}
}

var ClusterInstInfoApiCmds = []*cobra.Command{
	ShowClusterInstInfoCmd.GenCmd(),
}

var ClusterInstKeyV1RequiredArgs = []string{}
var ClusterInstKeyV1OptionalArgs = []string{
	"clusterkey.name",
	"cloudletkey.organization",
	"cloudletkey.name",
	"cloudletkey.federatedorganization",
	"organization",
}
var ClusterInstKeyV1AliasArgs = []string{}
var ClusterInstKeyV1Comments = map[string]string{
	"clusterkey.name":                   "Cluster name",
	"cloudletkey.organization":          "Organization of the cloudlet site",
	"cloudletkey.name":                  "Name of the cloudlet",
	"cloudletkey.federatedorganization": "Federated operator organization who shared this cloudlet",
	"organization":                      "Name of Developer organization that this cluster belongs to",
}
var ClusterInstKeyV1SpecialArgs = map[string]string{}
var ClusterInstKeyV2RequiredArgs = []string{}
var ClusterInstKeyV2OptionalArgs = []string{
	"clusterkey.name",
	"clusterkey.organization",
	"cloudletkey.organization",
	"cloudletkey.name",
	"cloudletkey.federatedorganization",
}
var ClusterInstKeyV2AliasArgs = []string{}
var ClusterInstKeyV2Comments = map[string]string{
	"clusterkey.name":                   "Cluster name",
	"clusterkey.organization":           "Name of the organization that this cluster belongs to",
	"cloudletkey.organization":          "Organization of the cloudlet site",
	"cloudletkey.name":                  "Name of the cloudlet",
	"cloudletkey.federatedorganization": "Federated operator organization who shared this cloudlet",
}
var ClusterInstKeyV2SpecialArgs = map[string]string{}
var ClusterInstRequiredArgs = []string{
	"cluster",
	"clusterorg",
}
var ClusterInstOptionalArgs = []string{
	"zoneorg",
	"zone",
	"zonekey.federatedorganization",
	"flavor",
	"crmoverride",
	"ipaccess",
	"deployment",
	"nummasters",
	"numnodes",
	"autoscalepolicy",
	"imagename",
	"reservable",
	"sharedvolumesize",
	"skipcrmcleanuponfailure",
	"multitenant",
	"networks",
	"enableipv6",
	"objid",
	"annotations",
	"dbmodelid",
	"noderesources.vcpus",
	"noderesources.ram",
	"noderesources.disk",
	"noderesources.optresmap",
	"noderesources.infranodeflavor",
	"noderesources.externalvolumesize",
	"nodepools:empty",
	"nodepools:#.name",
	"nodepools:#.numnodes",
	"nodepools:#.noderesources.vcpus",
	"nodepools:#.noderesources.ram",
	"nodepools:#.noderesources.disk",
	"nodepools:#.noderesources.optresmap",
	"nodepools:#.noderesources.infranodeflavor",
	"nodepools:#.noderesources.externalvolumesize",
	"nodepools:#.scalable",
	"infraannotations",
	"kubernetesversion",
	"disabledynamicappinstplacement",
	"tags",
}
var ClusterInstAliasArgs = []string{
	"cluster=key.name",
	"clusterorg=key.organization",
	"cloudletorg=cloudletkey.organization",
	"cloudlet=cloudletkey.name",
	"federatedorg=cloudletkey.federatedorganization",
	"zoneorg=zonekey.organization",
	"zone=zonekey.name",
	"flavor=flavor.name",
}
var ClusterInstComments = map[string]string{
	"fields":                            "Fields are used for the Update API to specify which fields to apply",
	"cluster":                           "Cluster name",
	"clusterorg":                        "Name of the organization that this cluster belongs to",
	"cloudletorg":                       "Organization of the cloudlet site",
	"cloudlet":                          "Name of the cloudlet",
	"federatedorg":                      "Federated operator organization who shared this cloudlet",
	"zoneorg":                           "Organization owner of the Zone",
	"zone":                              "Name of the Zone",
	"zonekey.federatedorganization":     "Federated operator organization who shared this Zone",
	"flavor":                            "Flavor name",
	"liveness":                          "Liveness of instance (see Liveness), one of Unknown, Static, Dynamic, Autoprov",
	"auto":                              "Auto is set to true when automatically created by back-end (internal use only)",
	"state":                             "State of the cluster instance, one of TrackedStateUnknown, NotPresent, CreateRequested, Creating, CreateError, Ready, UpdateRequested, Updating, UpdateError, DeleteRequested, Deleting, DeleteError, DeletePrepare, CrmInitok, CreatingDependencies, DeleteDone",
	"errors":                            "Any errors trying to create, update, or delete the ClusterInst on the Cloudlet., specify errors:empty=true to clear",
	"crmoverride":                       "Override actions to CRM, one of NoOverride, IgnoreCrmErrors, IgnoreCrm, IgnoreTransientState, IgnoreCrmAndTransientState",
	"ipaccess":                          "IP access type (RootLB Type), one of Unknown, Dedicated, Shared",
	"allocatedip":                       "Allocated IP for dedicated access",
	"nodeflavor":                        "(_deprecated_) Replaced by NodePools.NodeResources.InfraNodeFlavor. Cloudlet specific node flavor.",
	"deployment":                        "Deployment type (kubernetes or docker)",
	"nummasters":                        "Number of k8s masters (In case of docker deployment, this field is not required)",
	"numnodes":                          "(_deprecated_) Replaced by NodePools.NumNodes. Number of k8s nodes.",
	"externalvolumesize":                "(_deprecated_) Replaced by NodePools.NodeResources.ExternalVolumeSize. Size of external volume to be attached to nodes. This is for the root partition",
	"autoscalepolicy":                   "Auto scale policy name",
	"availabilityzone":                  "Optional Resource AZ if any",
	"imagename":                         "Optional resource specific image to launch",
	"reservable":                        "If ClusterInst is reservable",
	"reservedby":                        "For reservable EdgeCloud ClusterInsts, the current developer tenant",
	"sharedvolumesize":                  "Size of an optional shared volume to be mounted on the master",
	"masternodeflavor":                  "Generic flavor for k8s master VM when worker nodes > 0",
	"skipcrmcleanuponfailure":           "Prevents cleanup of resources on failure within CRM, used for diagnostic purposes",
	"optres":                            "Optional Resources required by OS flavor if any",
	"resources.vms:empty":               "Virtual machine resources info, specify resources.vms:empty=true to clear",
	"resources.vms:#.name":              "Virtual machine name",
	"resources.vms:#.type":              "Type can be platformvm, platform-cluster-master, platform-cluster-primary-node, platform-cluster-secondary-node, sharedrootlb, dedicatedrootlb, cluster-master, cluster-k8s-node, cluster-docker-node, appvm",
	"resources.vms:#.status":            "Runtime status of the VM",
	"resources.vms:#.infraflavor":       "Flavor allocated within the cloudlet infrastructure, distinct from the control plane flavor",
	"resources.vms:#.ipaddresses:empty": "IP addresses allocated to the VM, specify resources.vms:#.ipaddresses:empty=true to clear",
	"resources.vms:#.ipaddresses:#.externalip":     "External IP address",
	"resources.vms:#.ipaddresses:#.internalip":     "Internal IP address",
	"resources.vms:#.containers:empty":             "Information about containers running in the VM, specify resources.vms:#.containers:empty=true to clear",
	"resources.vms:#.containers:#.name":            "Name of the container",
	"resources.vms:#.containers:#.type":            "Type can be docker or kubernetes",
	"resources.vms:#.containers:#.status":          "Runtime status of the container",
	"resources.vms:#.containers:#.clusterip":       "IP within the CNI and is applicable to kubernetes only",
	"resources.vms:#.containers:#.restarts":        "Restart count, applicable to kubernetes only",
	"createdat":                                    "Created at time",
	"updatedat":                                    "Updated at time",
	"reservationendedat":                           "For reservable ClusterInsts, when the last reservation ended",
	"multitenant":                                  "Multi-tenant kubernetes cluster",
	"networks":                                     "networks to connect to, specify networks:empty=true to clear",
	"deleteprepare":                                "Preparing to be deleted",
	"dnslabel":                                     "DNS label that is unique within the cloudlet and among other AppInsts/ClusterInsts",
	"fqdn":                                         "FQDN is a globally unique DNS id for the ClusterInst",
	"staticfqdn":                                   "Static startup FQDN is an immutable globally unique name, set when object is created",
	"enableipv6":                                   "Enable IPv6 addressing, requires platform and cloudlet support, defaults to platform setting",
	"objid":                                        "Universally unique object ID",
	"compatibilityversion":                         "internal compatibility version",
	"annotations":                                  "Annotations, specify annotations:empty=true to clear",
	"dbmodelid":                                    "database version model ID",
	"noderesources.vcpus":                          "Vcpus to be allocated to the VM, must be either 1 or an even number",
	"noderesources.ram":                            "Total RAM in megabytes to be allocated to the VM",
	"noderesources.disk":                           "Total disk space in gigabytes to be allocated to the VMs root partition",
	"noderesources.optresmap":                      "Optional resources request, key = gpu form: $resource=$kind:[$alias]$count ex: optresmap=gpu=vgpu:nvidia-63:1, specify noderesources.optresmap:empty=true to clear",
	"noderesources.infranodeflavor":                "Infrastructure specific node flavor",
	"noderesources.externalvolumesize":             "Size of external volume to be attached to nodes. This is for the root partition",
	"nodepools:empty":                              "Node Pool specifications for Kubernetes clusters, specify nodepools:empty=true to clear",
	"nodepools:#.name":                             "Node pool name",
	"nodepools:#.numnodes":                         "Number of nodes in the pool",
	"nodepools:#.noderesources.vcpus":              "Vcpus to be allocated to the VM, must be either 1 or an even number",
	"nodepools:#.noderesources.ram":                "Total RAM in megabytes to be allocated to the VM",
	"nodepools:#.noderesources.disk":               "Total disk space in gigabytes to be allocated to the VMs root partition",
	"nodepools:#.noderesources.optresmap":          "Optional resources request, key = gpu form: $resource=$kind:[$alias]$count ex: optresmap=gpu=vgpu:nvidia-63:1, specify nodepools:#.noderesources.optresmap:empty=true to clear",
	"nodepools:#.noderesources.infranodeflavor":    "Infrastructure specific node flavor",
	"nodepools:#.noderesources.externalvolumesize": "Size of external volume to be attached to nodes. This is for the root partition",
	"nodepools:#.scalable":                         "Scalable indicates the system may scale the number of nodes",
	"infraannotations":                             "Annotations added by the implementing infrastructure, specify infraannotations:empty=true to clear",
	"kubernetesversion":                            "Kubernetes version of cluster if applicable",
	"disabledynamicappinstplacement":               "Disables dynamic placement of AppInsts on this cluster",
	"tags":                                         "Vendor-specific data, specify tags:empty=true to clear",
}
var ClusterInstSpecialArgs = map[string]string{
	"annotations":                         "StringToString",
	"errors":                              "StringArray",
	"fields":                              "StringArray",
	"infraannotations":                    "StringToString",
	"networks":                            "StringArray",
	"nodepools:#.noderesources.optresmap": "StringToString",
	"noderesources.optresmap":             "StringToString",
	"tags":                                "StringToString",
}
var IdleReservableClusterInstsRequiredArgs = []string{}
var IdleReservableClusterInstsOptionalArgs = []string{
	"idletime",
}
var IdleReservableClusterInstsAliasArgs = []string{}
var IdleReservableClusterInstsComments = map[string]string{
	"idletime": "Idle time (duration)",
}
var IdleReservableClusterInstsSpecialArgs = map[string]string{}
var ClusterResourceUsageRequiredArgs = []string{
	"cluster",
	"clusterorg",
}
var ClusterResourceUsageOptionalArgs = []string{
	"zonekey.organization",
	"zonekey.name",
	"zonekey.federatedorganization",
	"cloudletkey.organization",
	"cloudletkey.name",
	"cloudletkey.federatedorganization",
	"totalresources:#.name",
	"totalresources:#.value",
	"totalresources:#.inframaxvalue",
	"totalresources:#.quotamaxvalue",
	"totalresources:#.description",
	"totalresources:#.units",
	"totalresources:#.alertthreshold",
	"cpupoolsresources:#.name",
	"cpupoolsresources:#.value",
	"cpupoolsresources:#.inframaxvalue",
	"cpupoolsresources:#.quotamaxvalue",
	"cpupoolsresources:#.description",
	"cpupoolsresources:#.units",
	"cpupoolsresources:#.alertthreshold",
	"gpupoolsresources:#.name",
	"gpupoolsresources:#.value",
	"gpupoolsresources:#.inframaxvalue",
	"gpupoolsresources:#.quotamaxvalue",
	"gpupoolsresources:#.description",
	"gpupoolsresources:#.units",
	"gpupoolsresources:#.alertthreshold",
	"resourcescore",
	"cpupoolsresourcescore",
	"gpupoolsresourcescore",
	"reservedby",
}
var ClusterResourceUsageAliasArgs = []string{
	"cluster=key.name",
	"clusterorg=key.organization",
}
var ClusterResourceUsageComments = map[string]string{
	"cluster":                            "Cluster name",
	"clusterorg":                         "Name of the organization that this cluster belongs to",
	"zonekey.organization":               "Organization owner of the Zone",
	"zonekey.name":                       "Name of the Zone",
	"zonekey.federatedorganization":      "Federated operator organization who shared this Zone",
	"cloudletkey.organization":           "Organization of the cloudlet site",
	"cloudletkey.name":                   "Name of the cloudlet",
	"cloudletkey.federatedorganization":  "Federated operator organization who shared this cloudlet",
	"totalresources:#.name":              "Resource name",
	"totalresources:#.value":             "Resource value",
	"totalresources:#.inframaxvalue":     "Resource infra max value",
	"totalresources:#.quotamaxvalue":     "Resource quota max value",
	"totalresources:#.description":       "Resource description",
	"totalresources:#.units":             "Resource units",
	"totalresources:#.alertthreshold":    "Generate alert when more than threshold percentage of resource is used",
	"cpupoolsresources:#.name":           "Resource name",
	"cpupoolsresources:#.value":          "Resource value",
	"cpupoolsresources:#.inframaxvalue":  "Resource infra max value",
	"cpupoolsresources:#.quotamaxvalue":  "Resource quota max value",
	"cpupoolsresources:#.description":    "Resource description",
	"cpupoolsresources:#.units":          "Resource units",
	"cpupoolsresources:#.alertthreshold": "Generate alert when more than threshold percentage of resource is used",
	"gpupoolsresources:#.name":           "Resource name",
	"gpupoolsresources:#.value":          "Resource value",
	"gpupoolsresources:#.inframaxvalue":  "Resource infra max value",
	"gpupoolsresources:#.quotamaxvalue":  "Resource quota max value",
	"gpupoolsresources:#.description":    "Resource description",
	"gpupoolsresources:#.units":          "Resource units",
	"gpupoolsresources:#.alertthreshold": "Generate alert when more than threshold percentage of resource is used",
	"resourcescore":                      "Resource score, higher score means more available resources",
	"cpupoolsresourcescore":              "CPU Pool Resource score for Kubernetes clusters, higher score means more available resources",
	"gpupoolsresourcescore":              "GPU Pool Resource score for Kubernetes clusters, higher score means more available resources",
	"reservedby":                         "For reservable clusters in use, the organization that has reserved the cluster",
}
var ClusterResourceUsageSpecialArgs = map[string]string{}
var ClusterInstInfoRequiredArgs = []string{
	"key.name",
	"key.organization",
}
var ClusterInstInfoOptionalArgs = []string{
	"notifyid",
	"state",
	"errors",
	"status.tasknumber",
	"status.maxtasks",
	"status.taskname",
	"status.stepname",
	"status.msgcount",
	"status.msgs",
	"resources.vms:#.name",
	"resources.vms:#.type",
	"resources.vms:#.status",
	"resources.vms:#.infraflavor",
	"resources.vms:#.ipaddresses:#.externalip",
	"resources.vms:#.ipaddresses:#.internalip",
	"resources.vms:#.containers:#.name",
	"resources.vms:#.containers:#.type",
	"resources.vms:#.containers:#.status",
	"resources.vms:#.containers:#.clusterip",
	"resources.vms:#.containers:#.restarts",
	"infraannotations",
}
var ClusterInstInfoAliasArgs = []string{}
var ClusterInstInfoComments = map[string]string{
	"fields":                      "Fields are used for the Update API to specify which fields to apply",
	"key.name":                    "Cluster name",
	"key.organization":            "Name of the organization that this cluster belongs to",
	"notifyid":                    "Id of client assigned by server (internal use only)",
	"state":                       "State of the cluster instance, one of TrackedStateUnknown, NotPresent, CreateRequested, Creating, CreateError, Ready, UpdateRequested, Updating, UpdateError, DeleteRequested, Deleting, DeleteError, DeletePrepare, CrmInitok, CreatingDependencies, DeleteDone",
	"errors":                      "Any errors trying to create, update, or delete the ClusterInst on the Cloudlet.",
	"status.tasknumber":           "Task number",
	"status.maxtasks":             "Max tasks",
	"status.taskname":             "Task name",
	"status.stepname":             "Step name",
	"status.msgcount":             "Message count",
	"status.msgs":                 "Messages",
	"resources.vms:#.name":        "Virtual machine name",
	"resources.vms:#.type":        "Type can be platformvm, platform-cluster-master, platform-cluster-primary-node, platform-cluster-secondary-node, sharedrootlb, dedicatedrootlb, cluster-master, cluster-k8s-node, cluster-docker-node, appvm",
	"resources.vms:#.status":      "Runtime status of the VM",
	"resources.vms:#.infraflavor": "Flavor allocated within the cloudlet infrastructure, distinct from the control plane flavor",
	"resources.vms:#.ipaddresses:#.externalip": "External IP address",
	"resources.vms:#.ipaddresses:#.internalip": "Internal IP address",
	"resources.vms:#.containers:#.name":        "Name of the container",
	"resources.vms:#.containers:#.type":        "Type can be docker or kubernetes",
	"resources.vms:#.containers:#.status":      "Runtime status of the container",
	"resources.vms:#.containers:#.clusterip":   "IP within the CNI and is applicable to kubernetes only",
	"resources.vms:#.containers:#.restarts":    "Restart count, applicable to kubernetes only",
	"infraannotations":                         "Annotations added by the implementing infrastructure",
}
var ClusterInstInfoSpecialArgs = map[string]string{
	"errors":           "StringArray",
	"fields":           "StringArray",
	"infraannotations": "StringToString",
	"status.msgs":      "StringArray",
}
var UpdateClusterInstRequiredArgs = []string{
	"cluster",
	"clusterorg",
}
var UpdateClusterInstOptionalArgs = []string{
	"zoneorg",
	"zone",
	"zonekey.federatedorganization",
	"crmoverride",
	"numnodes",
	"autoscalepolicy",
	"skipcrmcleanuponfailure",
	"enableipv6",
	"objid",
	"annotations",
	"dbmodelid",
	"nodepools:empty",
	"nodepools:#.numnodes",
	"nodepools:#.scalable",
	"infraannotations",
	"disabledynamicappinstplacement",
	"tags",
}
var ShowClusterResourceUsageRequiredArgs = []string{}
var ShowClusterResourceUsageOptionalArgs = []string{
	"cluster",
	"clusterorg",
	"zoneorg",
	"zone",
	"zonekey.federatedorganization",
	"flavor",
	"crmoverride",
	"ipaccess",
	"deployment",
	"nummasters",
	"numnodes",
	"autoscalepolicy",
	"imagename",
	"reservable",
	"sharedvolumesize",
	"skipcrmcleanuponfailure",
	"multitenant",
	"networks",
	"enableipv6",
	"objid",
	"annotations",
	"dbmodelid",
	"noderesources.vcpus",
	"noderesources.ram",
	"noderesources.disk",
	"noderesources.optresmap",
	"noderesources.infranodeflavor",
	"noderesources.externalvolumesize",
	"nodepools:#.name",
	"nodepools:#.numnodes",
	"nodepools:#.noderesources.vcpus",
	"nodepools:#.noderesources.ram",
	"nodepools:#.noderesources.disk",
	"nodepools:#.noderesources.optresmap",
	"nodepools:#.noderesources.infranodeflavor",
	"nodepools:#.noderesources.externalvolumesize",
	"nodepools:#.scalable",
	"infraannotations",
	"kubernetesversion",
	"disabledynamicappinstplacement",
	"tags",
}

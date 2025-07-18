package proxmox

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	pveSDK "github.com/Telmate/proxmox-api-go/proxmox"
	"github.com/Telmate/terraform-provider-proxmox/v2/proxmox/Internal/resource/guest/node"
	"github.com/Telmate/terraform-provider-proxmox/v2/proxmox/Internal/resource/guest/pool"
	"github.com/Telmate/terraform-provider-proxmox/v2/proxmox/Internal/resource/guest/tags"
	vmID "github.com/Telmate/terraform-provider-proxmox/v2/proxmox/Internal/resource/guest/vmid"
	"github.com/Telmate/terraform-provider-proxmox/v2/proxmox/Internal/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var lxcResourceDef *schema.Resource

// TODO update tag schema
func resourceLxc() *schema.Resource {
	lxcResourceDef = &schema.Resource{
		CreateContext: resourceLxcCreate,
		ReadContext:   resourceLxcRead,
		UpdateContext: resourceLxcUpdate,
		DeleteContext: resourceVmQemuDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"ostemplate": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"arch": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "amd64",
			},
			"bwlimit": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"clone": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"clone_storage": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"cmode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "tty",
			},
			"console": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"cores": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cpulimit": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"cpuunits": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1024,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"features": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"fuse": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"keyctl": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"mknod": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"mount": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"nesting": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"full": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"force": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"hastate": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"hagroup": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"hookscript": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ignore_unpack_errors": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"lock": {
				Type:     schema.TypeString,
				Optional: true,
			},
			tags.Root: tags.Schema(),
			"memory": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  512,
			},
			"mountpoint": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Total Hackery here. A TypeMap would be amazing if it supported Resources as values...
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"slot": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"mp": {
							Type:     schema.TypeString,
							Required: true,
						},
						"storage": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"acl": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"backup": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"quota": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"replicate": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"shared": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"size": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(string)
								if !(strings.Contains(v, "T") || strings.Contains(v, "G") || strings.Contains(v, "M") || strings.Contains(v, "K")) {
									errs = append(errs, fmt.Errorf("disk size must end in T, G, M, or K, got %s", v))
								}
								return
							},
						},
						"volume": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"file": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"nameserver": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"bridge": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"firewall": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"gw": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"gw6": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"hwaddr": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"ip": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip6": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"mtu": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"rate": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"tag": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"trunks": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"onboot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ostype": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				ForceNew:  true, // Proxmox doesn't support password changes
			},
			pool.Root: pool.Schema(),
			"protection": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"restore": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"rootfs": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"acl": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"quota": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"replicate": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"ro": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"shared": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"storage": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"size": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(string)
								if !(strings.Contains(v, "T") || strings.Contains(v, "G") || strings.Contains(v, "M") || strings.Contains(v, "K")) {
									errs = append(errs, fmt.Errorf("disk size must end in T, G, M, or K, got %s", v))
								}
								return
							},
						},
						"volume": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"searchdomain": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_public_keys": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"start": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"startup": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"swap": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"template": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"tty": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
			"unique": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"unprivileged": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"unused": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			node.Computed: node.SchemaComputed("qemu"),
			node.RootNode: node.SchemaNode(schema.Schema{}, "lxc"),
			vmID.Root:     vmID.Schema(),
		},
		Timeouts: resourceTimeouts(),
	}

	return lxcResourceDef
}

func resourceLxcCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pconf := meta.(*providerConfiguration)

	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	client := pconf.Client

	config := pveSDK.NewConfigLxc()
	config.Ostemplate = d.Get("ostemplate").(string)
	config.Arch = d.Get("arch").(string)
	config.BWLimit = d.Get("bwlimit").(int)
	config.Clone = d.Get("clone").(string)
	config.CloneStorage = d.Get("clone_storage").(string)
	config.CMode = d.Get("cmode").(string)
	config.Console = d.Get("console").(bool)
	config.Cores = d.Get("cores").(int)
	config.CPULimit = d.Get("cpulimit").(int)
	config.CPUUnits = d.Get("cpuunits").(int)
	config.Description = d.Get("description").(string)
	features := d.Get("features").(*schema.Set)
	featureSetList := features.List()
	if len(featureSetList) > 0 {
		// only apply the first feature set,
		// because proxmox api only allows one feature set
		config.Features = featureSetList[0].(map[string]interface{})
	}
	config.Force = d.Get("force").(bool)
	config.HaState = d.Get("hastate").(string)
	config.HaGroup = d.Get("hagroup").(string)
	config.Hookscript = d.Get("hookscript").(string)
	config.Hostname = d.Get("hostname").(string)
	config.IgnoreUnpackErrors = d.Get("ignore_unpack_errors").(bool)
	config.Lock = d.Get("lock").(string)
	config.Memory = d.Get("memory").(int)
	config.Nameserver = d.Get("nameserver").(string)
	config.OnBoot = d.Get("onboot").(bool)
	config.OsType = d.Get("ostype").(string)
	config.Password = d.Get("password").(string)
	config.Pool = util.Pointer(pveSDK.PoolName(d.Get(pool.Root).(string)))
	config.Protection = d.Get("protection").(bool)
	config.Restore = d.Get("restore").(bool)
	config.SearchDomain = d.Get("searchdomain").(string)
	config.SSHPublicKeys = d.Get("ssh_public_keys").(string)
	config.Start = d.Get("start").(bool)
	config.Startup = d.Get("startup").(string)
	config.Swap = d.Get("swap").(int)
	config.Tags = tags.SDK(d).String()
	config.Template = d.Get("template").(bool)
	config.Tty = d.Get("tty").(int)
	config.Unique = d.Get("unique").(bool)
	config.Unprivileged = d.Get("unprivileged").(bool)

	// proxmox api allows multiple network sets,
	// having a unique 'id' parameter foreach set
	networks := d.Get("network").([]interface{})
	if len(networks) > 0 {
		lxcNetworks := DevicesListToDevices(networks, "")
		config.Networks = lxcNetworks
	}

	rootfs, exists := d.GetOk("rootfs")

	if exists {
		config.RootFs = rootfs.([]interface{})[0].(map[string]interface{})
	}

	// proxmox api allows multiple mountpoint sets,
	// having a unique 'id' parameter foreach set
	mountpoints := d.Get("mountpoint").([]interface{})
	if len(mountpoints) > 0 {
		lxcMountpoints := DevicesListToDevices(mountpoints, "slot")
		config.Mountpoints = lxcMountpoints
	}

	// get unique id

	setGuestID := d.Get(vmID.Root).(int)

	targetNode, err := node.SdkCreate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	var diags diag.Diagnostics
	var vmr *pveSDK.VmRef
	if d.Get("clone").(string) != "" { // Clone
		var sourceVmr *pveSDK.VmRef
		sourceVmr, err = getSourceVmr(ctx, client, pveSDK.GuestName(d.Get("clone").(string)), 0, targetNode)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}

		var guestID *pveSDK.GuestID
		if setGuestID != 0 {
			guestID = util.Pointer(pveSDK.GuestID(setGuestID))
		}

		var storage *string
		if config.CloneStorage != "" {
			storage = &config.CloneStorage
		}

		var poolName *pveSDK.PoolName
		tmpPool := d.Get(pool.Root).(string)
		if tmpPool != "" {
			poolName = util.Pointer(pveSDK.PoolName(tmpPool))
		}

		hostname := pveSDK.GuestName(config.Hostname)
		var cloneSettings pveSDK.CloneLxcTarget
		if !d.Get("full").(bool) {
			cloneSettings = pveSDK.CloneLxcTarget{
				Linked: &pveSDK.CloneLinked{
					Node: targetNode,
					ID:   guestID,
					Name: &hostname,
					Pool: poolName}}
		} else {
			cloneSettings = pveSDK.CloneLxcTarget{
				Full: &pveSDK.CloneLxcFull{
					Node:    targetNode,
					ID:      guestID,
					Name:    &hostname,
					Pool:    poolName,
					Storage: storage}}
		}

		log.Print("[DEBUG][LxcCreate] cloning LXC")
		vmr, err = sourceVmr.CloneLxc(ctx, cloneSettings, client)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}

		// Waiting for the clone to become ready and
		// read back all the current disk configurations from proxmox
		// this allows us to receive updates on the post-clone state of the container we're building
		log.Print("[DEBUG][LxcCreate] Waiting for clone becoming ready")
		var config_post_clone *pveSDK.ConfigLxc
		for {
			// Wait until we can actually retrieve the config from the cloned machine
			config_post_clone, err = pveSDK.NewConfigLxcFromApi(ctx, vmr, client)
			if config_post_clone != nil {
				break
				// to prevent an infinite loop we check for any other error
				// this error is actually fine because the clone is not ready yet
			} else if err.Error() != "vm locked, could not obtain config" {
				return append(diags, diag.FromErr(err)...)
			}
			time.Sleep(5 * time.Second)
			log.Print("[DEBUG][LxcCreate] Clone still not ready, checking again")
		}
		if config_post_clone.RootFs["size"] == config.RootFs["size"] {
			log.Print("[DEBUG][LxcCreate] Waiting for clone becoming ready")
		} else {
			log.Print("[DEBUG][LxcCreate] We must resize")
			processDiskResize(ctx, config_post_clone.RootFs, config.RootFs, "rootfs", pconf, vmr)
		}
		config_post_resize, err := pveSDK.NewConfigLxcFromApi(ctx, vmr, client)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}
		config.RootFs["size"] = config_post_resize.RootFs["size"]
		config.RootFs["volume"] = config_post_resize.RootFs["volume"]

		// Update all remaining stuff
		err = config.UpdateConfig(ctx, vmr, client)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}

	} else { // Create
		nextID := pveSDK.GuestID(setGuestID)
		if setGuestID == 0 {
			if pconf.MaxParallel > 1 { // TODO actually fix the parallelism! workaround for #1136
				diags = append(diags, diag.Diagnostic{
					Summary:  "setting " + schemaPmParallel + " greater than 1 is currently not recommended when creating LXC containers with dynamic id allocation",
					Severity: diag.Warning})
			}

			nextID, err = nextVmId(pconf)
			if err != nil {
				return append(diags, diag.FromErr(err)...)
			}
		}

		vmr = pveSDK.NewVmRef(nextID)
		vmr.SetNode(targetNode.String())

		err = config.CreateLxc(ctx, vmr, client)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}
	}

	// The existence of a non-blank ID is what tells Terraform that a resource was created
	d.SetId(resourceId(targetNode, "lxc", vmr.VmId()))

	lock.unlock()
	return append(diags, resourceLxcRead(ctx, d, meta)...)
}

func resourceLxcUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()

	client := pconf.Client

	_, _, vmID, err := parseResourceId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	vmr := pveSDK.NewVmRef(vmID)
	_, err = client.GetVmInfo(ctx, vmr)
	if err != nil {
		return diag.FromErr(err)
	}

	config := pveSDK.NewConfigLxc()
	config.Ostemplate = d.Get("ostemplate").(string)
	config.Arch = d.Get("arch").(string)
	config.BWLimit = d.Get("bwlimit").(int)
	config.CMode = d.Get("cmode").(string)
	config.Console = d.Get("console").(bool)
	config.Cores = d.Get("cores").(int)
	config.CPULimit = d.Get("cpulimit").(int)
	config.CPUUnits = d.Get("cpuunits").(int)
	config.Description = d.Get("description").(string)
	features := d.Get("features").(*schema.Set)
	featureSetList := features.List()
	if len(featureSetList) > 0 {
		// only apply the first feature set,
		// because proxmox api only allows one feature set
		config.Features = featureSetList[0].(map[string]interface{})
	}
	config.Force = d.Get("force").(bool)
	config.HaState = d.Get("hastate").(string)
	config.HaGroup = d.Get("hagroup").(string)
	config.Hookscript = d.Get("hookscript").(string)
	config.Hostname = d.Get("hostname").(string)
	config.IgnoreUnpackErrors = d.Get("ignore_unpack_errors").(bool)
	config.Lock = d.Get("lock").(string)
	config.Memory = d.Get("memory").(int)
	config.Nameserver = d.Get("nameserver").(string)
	config.OnBoot = d.Get("onboot").(bool)
	config.OsType = d.Get("ostype").(string)
	config.Password = d.Get("password").(string)
	config.Pool = util.Pointer(pveSDK.PoolName(d.Get(pool.Root).(string)))
	config.Protection = d.Get("protection").(bool)
	config.Restore = d.Get("restore").(bool)
	config.SearchDomain = d.Get("searchdomain").(string)
	config.Start = d.Get("start").(bool)
	config.Startup = d.Get("startup").(string)
	config.Swap = d.Get("swap").(int)
	config.Tags = tags.SDK(d).String()
	config.Template = d.Get("template").(bool)
	config.Tty = d.Get("tty").(int)
	config.Unique = d.Get("unique").(bool)
	config.Unprivileged = d.Get("unprivileged").(bool)

	if d.HasChange("network") {
		oldSet, newSet := d.GetChange("network")
		oldNetworks := make([]map[string]interface{}, 0)
		newNetworks := make([]map[string]interface{}, 0)

		// Convert from []interface{} to []map[string]interface{}
		for _, network := range oldSet.([]interface{}) {
			oldNetworks = append(oldNetworks, network.(map[string]interface{}))
		}
		for _, network := range newSet.([]interface{}) {
			newNetworks = append(newNetworks, network.(map[string]interface{}))
		}

		processLxcNetworkChanges(ctx, oldNetworks, newNetworks, pconf, vmr)

		if len(newNetworks) > 0 {
			// Drop all the ids since they can't be sent to the API
			newNetworks, _ = DropElementsFromMap([]string{"id"}, newNetworks)
			// Convert from []map[string]interface{} to pxapi.QemuDevices
			lxcNetworks := make(pveSDK.QemuDevices, 0)
			for index, network := range newNetworks {
				lxcNetworks[index] = network
			}
			config.Networks = lxcNetworks
		}
	}

	_, exists := d.GetOk("rootfs")

	if exists && d.HasChange("rootfs") {
		oldSet, newSet := d.GetChange("rootfs")

		oldRootFs := oldSet.([]interface{})[0].(map[string]interface{})
		newRootFs := newSet.([]interface{})[0].(map[string]interface{})

		processLxcDiskChanges(ctx, DeviceToMap(oldRootFs, 0), DeviceToMap(newRootFs, 0), pconf, vmr)
		config.RootFs = newRootFs
	}

	if d.HasChange("mountpoint") {
		oldSet, newSet := d.GetChange("mountpoint")
		oldMounts := DevicesListToMapByKey(oldSet.([]interface{}), "key")
		newMounts := DevicesListToMapByKey(newSet.([]interface{}), "key")
		processLxcDiskChanges(ctx, oldMounts, newMounts, pconf, vmr)

		lxcMountpoints := DevicesListToDevices(newSet.([]interface{}), "slot")
		config.Mountpoints = lxcMountpoints
	}

	// TODO: Detect changes requiring Reboot

	err = config.UpdateConfig(ctx, vmr, client)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(pool.Root) {
		oldPool, newPool := func() (string, string) {
			a, b := d.GetChange(pool.Root)
			return a.(string), b.(string)
		}()

		vmr := pveSDK.NewVmRef(vmID)
		vmr.SetPool(oldPool)

		_, err := client.UpdateVMPool(ctx, vmr, newPool)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("start") {
		guestState, err := vmr.GetRawGuestStatus(ctx, client)
		if err != nil {
			return diag.Diagnostics{{
				Summary:  err.Error(),
				Severity: diag.Error}}
		}
		switch guestState.State() {
		case pveSDK.PowerStateStopped:
			if d.Get("start").(bool) {
				log.Print("[DEBUG][LXCUpdate] starting LXC")
				_, err = client.StartVm(ctx, vmr)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		case pveSDK.PowerStateRunning:
			if !d.Get("start").(bool) {
				log.Print("[DEBUG][LXCUpdate] stopping LXC")
				_, err = client.StopVm(ctx, vmr)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	lock.unlock()
	return resourceLxcRead(ctx, d, meta)
}

func resourceLxcRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.FromErr(_resourceLxcRead(ctx, d, meta))
}

func _resourceLxcRead(ctx context.Context, d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	lock := pmParallelBegin(pconf)
	defer lock.unlock()
	client := pconf.Client
	_, _, guestID, err := parseResourceId(d.Id())
	if err != nil {
		d.SetId("")
		return err
	}
	vmr := pveSDK.NewVmRef(guestID)
	_, err = client.GetVmInfo(ctx, vmr)
	if err != nil {
		return err
	}
	config, err := pveSDK.NewConfigLxcFromApi(ctx, vmr, client)
	if err != nil {
		return err
	}
	d.SetId(resourceId(vmr.Node(), "lxc", vmr.VmId()))
	node.Terraform(vmr.Node(), d)

	// Read Features
	defaultFeatures := d.Get("features").(*schema.Set)
	if len(defaultFeatures.List()) > 0 {
		featuresWithDefaults := UpdateDeviceConfDefaults(config.Features, defaultFeatures)
		d.Set("features", featuresWithDefaults)
	}

	// Read Mountpoints
	configMountpointSet := d.Get("mountpoint").([]interface{})
	configMountpointMap := DevicesListToMapByKey(configMountpointSet, "slot")
	if len(configMountpointSet) > 0 {
		for slot, device := range config.Mountpoints {
			if confDevice, ok := configMountpointMap[slot]; ok {
				device["key"] = confDevice["key"]
			}
		}

		if err = AssertNoNonSchemaValues(config.Mountpoints, lxcResourceDef.Schema["mountpoint"]); err != nil {
			return err
		}

		flatMountpoints, _ := FlattenDevicesList(config.Mountpoints)
		if err = d.Set("mountpoint", flatMountpoints); err != nil {
			return err
		}
	}

	// Read RootFs
	rootFs := d.Get("rootfs").([]interface{})
	if len(rootFs) > 0 {
		confRootFs := rootFs[0]
		adaptedRootFs := adaptDeviceToConf(confRootFs.(map[string]interface{}), config.RootFs)
		d.Set("rootfs", []interface{}{adaptedRootFs})
	} else {
		confRootFs := make(map[string]interface{})
		confRootFs = adaptDeviceToConf(confRootFs, config.RootFs)
		adaptedRootFs := []map[string]interface{}{confRootFs}
		d.Set("rootfs", adaptedRootFs)
	}

	// Read Networks
	configNetworksSet := d.Get("network").([]interface{})
	if len(configNetworksSet) > 0 {
		if err = AssertNoNonSchemaValues(config.Networks, lxcResourceDef.Schema["network"]); err != nil {
			return err
		}
		flatNetworks, _ := FlattenDevicesList(config.Networks)
		if err = d.Set("network", flatNetworks); err != nil {
			return err
		}
	}

	// Pool
	pools, err := client.GetPoolList(ctx)
	if err == nil {
		for _, poolInfo := range pools["data"].([]interface{}) {
			poolContent, _ := client.GetPoolInfo(ctx, poolInfo.(map[string]interface{})["poolid"].(string))
			for _, member := range poolContent["members"].([]interface{}) {
				if member.(map[string]interface{})["type"] != "storage" {
					if guestID == pveSDK.GuestID(member.(map[string]interface{})[vmID.Root].(float64)) {
						d.Set(pool.Root, poolInfo.(map[string]interface{})["poolid"].(string))
					}
				}
			}
		}
	}

	//_, err = client.ReadVMHA(vmr)
	if err != nil {
		return err
	}
	d.Set("hastate", vmr.HaState())
	d.Set("hagroup", vmr.HaGroup())

	// Read Misc
	vmID.Terraform(vmr.VmId(), d)
	d.Set("arch", config.Arch)
	d.Set("bwlimit", config.BWLimit)
	d.Set("cmode", config.CMode)
	d.Set("console", config.Console)
	d.Set("cores", config.Cores)
	d.Set("cpulimit", config.CPULimit)
	d.Set("cpuunits", config.CPUUnits)
	d.Set("description", config.Description)
	d.Set("force", config.Force)

	d.Set("hookscript", config.Hookscript)
	d.Set("hostname", config.Hostname)
	d.Set("ignore_unpack_errors", config.IgnoreUnpackErrors)
	d.Set("lock", config.Lock)
	d.Set("memory", config.Memory)
	d.Set("nameserver", config.Nameserver)
	d.Set("onboot", config.OnBoot)
	d.Set("ostype", config.OsType)
	d.Set("protection", config.Protection)
	d.Set("restore", config.Restore)
	d.Set("searchdomain", config.SearchDomain)
	d.Set("startup", config.Startup)
	d.Set("swap", config.Swap)

	rawTags := strings.Split(config.Tags, ",")
	tmpTags := make(pveSDK.Tags, len(rawTags))
	for i := range rawTags {
		tmpTags[i] = pveSDK.Tag(rawTags[i])
	}
	tags.Terraform(&tmpTags, d)
	d.Set("template", config.Template)
	d.Set("tty", config.Tty)
	d.Set("unique", config.Unique)
	d.Set("unprivileged", config.Unprivileged)
	d.Set("unused", config.Unused)

	// Only applicable on create and not readable
	// d.Set("start", config.Start)
	// d.Set("ostemplate", config.Ostemplate)
	// d.Set("password", config.Password)
	// d.Set("ssh_public_keys", config.SSHPublicKeys)

	return nil
}

func processLxcDiskChanges(
	ctx context.Context,
	prevDiskSet KeyedDeviceMap, newDiskSet KeyedDeviceMap, pconf *providerConfiguration,
	vmr *pveSDK.VmRef,
) error {
	// 1. Delete slots that either a. Don't exist in the new set or b. Have a different volume in the new set
	deleteDisks := []pveSDK.QemuDevice{}
	for key, prevDisk := range prevDiskSet {
		newDisk, ok := (newDiskSet)[key]
		// The Rootfs can't be deleted
		if ok && diskSlotName(newDisk) == "rootfs" {
			continue
		}
		if !ok || (newDisk["volume"] != "" && prevDisk["volume"] != newDisk["volume"]) || (prevDisk["slot"] != newDisk["slot"]) {
			deleteDisks = append(deleteDisks, prevDisk)
		}
	}
	if len(deleteDisks) > 0 {
		deleteDiskKeys := []string{}
		for _, disk := range deleteDisks {
			deleteDiskKeys = append(deleteDiskKeys, diskSlotName(disk))
		}
		params := map[string]interface{}{}
		params["delete"] = strings.Join(deleteDiskKeys, ", ")
		if vmr.GetVmType() == "lxc" {
			if _, err := pconf.Client.SetLxcConfig(ctx, vmr, params); err != nil {
				return err
			}
		} else {
			if _, err := pconf.Client.SetVmConfig(vmr, params); err != nil {
				return err
			}
		}
	}

	// Create New Disks and Re-reference Slot-Changed Disks
	newParams := map[string]interface{}{}
	for key, newDisk := range newDiskSet {
		prevDisk, ok := prevDiskSet[key]
		diskName := diskSlotName(newDisk)

		if ok {
			for k, v := range prevDisk {
				_, ok := newDisk[k]
				if !ok {
					newDisk[k] = v
				}
			}
		}

		if !ok || newDisk["slot"] != prevDisk["slot"] {
			newParams[diskName] = pveSDK.FormatDiskParam(newDisk)
		}
	}
	if len(newParams) > 0 {
		if vmr.GetVmType() == "lxc" {
			if _, err := pconf.Client.SetLxcConfig(ctx, vmr, newParams); err != nil {
				return err
			}
		} else {
			if _, err := pconf.Client.SetVmConfig(vmr, newParams); err != nil {
				return err
			}
		}
	}

	// Move and Resize Existing Disks
	for key, prevDisk := range prevDiskSet {
		newDisk, ok := newDiskSet[key]
		diskName := diskSlotName(newDisk)
		if ok {
			// 2. Move disks with mismatching storage
			newStorage, ok := newDisk["storage"].(string)
			if ok && newStorage != prevDisk["storage"] {
				if vmr.GetVmType() == "lxc" {
					_, err := pconf.Client.MoveLxcDisk(ctx, vmr, diskSlotName(prevDisk), newStorage)
					if err != nil {
						return err
					}
				} else {
					_, err := pconf.Client.MoveQemuDisk(vmr, diskSlotName(prevDisk), newStorage)
					if err != nil {
						return err
					}
				}
			}

			// 3. Resize disks with different sizes
			if err := processDiskResize(ctx, prevDisk, newDisk, diskName, pconf, vmr); err != nil {
				return err
			}
		}
	}

	// Update Volume info
	apiResult, err := pconf.Client.GetVmConfig(ctx, vmr)
	if err != nil {
		return err
	}
	for _, newDisk := range newDiskSet {
		diskName := diskSlotName(newDisk)
		apiConfigStr := apiResult[diskName].(string)
		apiDevice := pveSDK.ParsePMConf(apiConfigStr, "volume")
		newDisk["volume"] = apiDevice["volume"]
	}

	return nil
}

func diskSlotName(disk pveSDK.QemuDevice) string {
	diskType, ok := disk["type"].(string)
	if !ok || diskType == "" {
		diskType = "mp"
	}
	diskSlot, ok := disk["slot"].(int)
	if !ok {
		return "rootfs"
	}
	return diskType + strconv.Itoa(diskSlot)
}

func processDiskResize(
	ctx context.Context,
	prevDisk pveSDK.QemuDevice, newDisk pveSDK.QemuDevice,
	diskName string,
	pconf *providerConfiguration, vmr *pveSDK.VmRef,
) error {
	newSize, ok := newDisk["size"]
	if ok && newSize != prevDisk["size"] {
		log.Print("[DEBUG][diskResize] resizing disk " + diskName)
		_, err := pconf.Client.ResizeQemuDiskRaw(ctx, vmr, diskName, newDisk["size"].(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func processLxcNetworkChanges(ctx context.Context, prevNetworks []map[string]interface{}, newNetworks []map[string]interface{}, pconf *providerConfiguration, vmr *pveSDK.VmRef) error {
	delNetworks := make([]map[string]interface{}, 0)

	// Collect the IDs of networks that exist in `prevNetworks` but not in `newNetworks`.
	for _, prevNet := range prevNetworks {
		found := false
		prevName := prevNet["id"].(int)
		for _, newNet := range newNetworks {
			newName := newNet["id"].(int)
			if prevName == newName {
				found = true
				break
			}
		}
		if !found {
			delNetworks = append(delNetworks, prevNet)
		}
	}

	if len(delNetworks) > 0 {
		deleteNetKeys := []string{}
		for _, net := range delNetworks {
			// Construct id that proxmox API expects (net+<number>)
			deleteNetKeys = append(deleteNetKeys, "net"+strconv.Itoa(net["id"].(int)))
		}

		params := map[string]interface{}{
			"delete": strings.Join(deleteNetKeys, ", "),
		}
		if vmr.GetVmType() == "lxc" {
			if _, err := pconf.Client.SetLxcConfig(ctx, vmr, params); err != nil {
				return err
			}
		} else {
			if _, err := pconf.Client.SetVmConfig(vmr, params); err != nil {
				return err
			}
		}
	}

	return nil
}

/*******************************************************************************
*
* Copyright 2017 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package keystone

import (
	"fmt"

	policy "github.com/databus23/goslo.policy"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/pkg/errors"
	"github.com/sapcc/hermes/pkg/util"
	"github.com/spf13/viper"
	"sync"
)

// Real keystone implementation
func Keystone() Driver {
	return keystone{}
}

type keystone struct {
	TokenRenewalMutex *sync.Mutex
}

var providerClient *gophercloud.ProviderClient
var domainNameCache *map[string]string
var projectNameCache *map[string]string
var userNameCache *map[string]string
var userIdCache *map[string]string
var roleNameCache *map[string]string
var groupNameCache *map[string]string

func (d keystone) keystoneClient() (*gophercloud.ServiceClient, error) {
	if d.TokenRenewalMutex == nil {
		d.TokenRenewalMutex = &sync.Mutex{}
	}
	if domainNameCache == nil {
		domainNameCache = &map[string]string{}
	}
	if projectNameCache == nil {
		projectNameCache = &map[string]string{}
	}
	if userNameCache == nil {
		userNameCache = &map[string]string{}
	}
	if userIdCache == nil {
		userIdCache = &map[string]string{}
	}
	if roleNameCache == nil {
		roleNameCache = &map[string]string{}
	}
	if groupNameCache == nil {
		groupNameCache = &map[string]string{}
	}
	if providerClient == nil {
		var err error
		providerClient, err = openstack.NewClient(viper.GetString("keystone.auth_url"))
		if err != nil {
			return nil, fmt.Errorf("cannot initialize OpenStack client: %v", err)
		}
		err = d.RefreshToken()
		if err != nil {
			return nil, fmt.Errorf("cannot fetch initial Keystone token: %v", err)
		}
	}

	return openstack.NewIdentityV3(providerClient,
		gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic},
	)
}

func (d keystone) Client() *gophercloud.ProviderClient {
	var kc keystone

	err := viper.UnmarshalKey("keystone", &kc)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	return nil
}

func (d keystone) ValidateToken(token string) (policy.Context, error) {
	client, err := d.keystoneClient()
	if err != nil {
		return policy.Context{}, err
	}

	response := tokens.Get(client, token)
	if response.Err != nil {
		//this includes 4xx responses, so after this point, we can be sure that the token is valid
		return policy.Context{}, response.Err
	}

	//use a custom token struct instead of tokens.Token which is way incomplete
	var tokenData keystoneToken
	err = response.ExtractInto(&tokenData)
	if err != nil {
		return policy.Context{}, err
	}
	d.updateCache(&tokenData)
	return tokenData.ToContext(), nil
}

func (d keystone) updateCache(token *keystoneToken) {
	if token.DomainScope.ID != "" && token.DomainScope.Name != "" {
		(*domainNameCache)[token.DomainScope.ID] = token.DomainScope.Name
	}
	if token.ProjectScope.Domain.ID != "" && token.ProjectScope.Domain.Name != "" {
		(*domainNameCache)[token.ProjectScope.Domain.ID] = token.ProjectScope.Domain.Name
	}
	if token.ProjectScope.ID != "" && token.ProjectScope.Name != "" {
		(*projectNameCache)[token.ProjectScope.ID] = token.ProjectScope.Name
	}
	if token.User.ID != "" && token.User.Name != "" {
		(*userNameCache)[token.User.ID] = token.User.Name
		(*userIdCache)[token.User.Name] = token.User.ID
	}
	for _, role := range token.Roles {
		if role.ID != "" && role.Name != "" {
			(*roleNameCache)[role.ID] = role.Name
		}
	}
}

func (d keystone) Authenticate(credentials *gophercloud.AuthOptions) (policy.Context, error) {
	client, err := d.keystoneClient()
	if err != nil {
		return policy.Context{}, err
	}
	response := tokens.Create(client, credentials)
	if response.Err != nil {
		//this includes 4xx responses, so after this point, we can be sure that the token is valid
		return policy.Context{}, response.Err
	}
	//use a custom token struct instead of tokens.Token which is way incomplete
	var tokenData keystoneToken
	err = response.ExtractInto(&tokenData)
	if err != nil {
		return policy.Context{}, err
	}
	return tokenData.ToContext(), nil
}

func (d keystone) DomainName(id string) (string, error) {
	cachedName, hit := (*domainNameCache)[id]
	if hit {
		return cachedName, nil
	}

	client, err := d.keystoneClient()
	if err != nil {
		return "", err
	}

	var result gophercloud.Result
	url := client.ServiceURL(fmt.Sprintf("domains/%s", id))
	_, err = client.Get(url, &result.Body, nil)
	if err != nil {
		return "", err
	}

	var data struct {
		Domain KeystoneNameId `json:"domain"`
	}
	err = result.ExtractInto(&data)
	if err == nil {
		(*domainNameCache)[id] = data.Domain.Name
	}
	return data.Domain.Name, err
}

func (d keystone) ProjectName(id string) (string, error) {
	cachedName, hit := (*projectNameCache)[id]
	if hit {
		return cachedName, nil
	}

	client, err := d.keystoneClient()
	if err != nil {
		return "", err
	}

	var result gophercloud.Result
	url := client.ServiceURL(fmt.Sprintf("projects/%s", id))
	_, err = client.Get(url, &result.Body, nil)
	if err != nil {
		return "", err
	}

	var data struct {
		Project KeystoneNameId `json:"project"`
	}
	err = result.ExtractInto(&data)
	if err == nil {
		(*projectNameCache)[id] = data.Project.Name
	}
	return data.Project.Name, err
}

func (d keystone) UserName(id string) (string, error) {
	cachedName, hit := (*userNameCache)[id]
	if hit {
		return cachedName, nil
	}

	client, err := d.keystoneClient()
	if err != nil {
		return "", err
	}

	var result gophercloud.Result
	url := client.ServiceURL(fmt.Sprintf("users/%s", id))
	_, err = client.Get(url, &result.Body, nil)
	if err != nil {
		return "", err
	}

	var data struct {
		User KeystoneNameId `json:"user"`
	}
	err = result.ExtractInto(&data)
	if err == nil {
		(*userNameCache)[id] = data.User.Name
		(*userIdCache)[data.User.Name] = id
	}
	return data.User.Name, err
}

func (d keystone) UserId(name string) (string, error) {
	cachedId, hit := (*userIdCache)[name]
	if hit {
		return cachedId, nil
	}

	client, err := d.keystoneClient()
	if err != nil {
		return "", err
	}

	var result gophercloud.Result
	url := client.ServiceURL(fmt.Sprintf("users?name=%s", name))
	_, err = client.Get(url, &result.Body, nil)
	if err != nil {
		return "", err
	}

	var data struct {
		User []KeystoneNameId `json:"user"`
	}
	err = result.ExtractInto(&data)
	userId := ""
	if err == nil {
		switch len(data.User) {
		case 0:
			err = errors.Errorf("No user found with name %s", name)
		case 1:
			userId = data.User[0].UUID
		default:
			util.LogWarning("Multiple users found with name %s - returning the first one", name)
			userId = data.User[0].UUID
		}
		(*userIdCache)[name] = userId
		(*userNameCache)[userId] = name
	}
	return userId, err
}

func (d keystone) RoleName(id string) (string, error) {
	cachedName, hit := (*roleNameCache)[id]
	if hit {
		return cachedName, nil
	}

	client, err := d.keystoneClient()
	if err != nil {
		return "", err
	}

	var result gophercloud.Result
	url := client.ServiceURL(fmt.Sprintf("roles/%s", id))
	_, err = client.Get(url, &result.Body, nil)
	if err != nil {
		return "", err
	}

	var data struct {
		Role KeystoneNameId `json:"role"`
	}
	err = result.ExtractInto(&data)
	if err == nil {
		(*roleNameCache)[id] = data.Role.Name
	}
	return data.Role.Name, err
}

func (d keystone) GroupName(id string) (string, error) {
	cachedName, hit := (*groupNameCache)[id]
	if hit {
		return cachedName, nil
	}

	client, err := d.keystoneClient()
	if err != nil {
		return "", err
	}

	var result gophercloud.Result
	url := client.ServiceURL(fmt.Sprintf("groups/%s", id))
	_, err = client.Get(url, &result.Body, nil)
	if err != nil {
		return "", err
	}

	var data struct {
		Group KeystoneNameId `json:"group"`
	}
	err = result.ExtractInto(&data)
	if err == nil {
		(*groupNameCache)[id] = data.Group.Name
	}
	return data.Group.Name, err
}

type keystoneToken struct {
	DomainScope  keystoneTokenThing         `json:"domain"`
	ProjectScope keystoneTokenThingInDomain `json:"project"`
	Roles        []keystoneTokenThing       `json:"roles"`
	User         keystoneTokenThingInDomain `json:"user"`
}

type keystoneTokenThing struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type keystoneTokenThingInDomain struct {
	keystoneTokenThing
	Domain keystoneTokenThing `json:"domain"`
}

func (t *keystoneToken) ToContext() policy.Context {
	c := policy.Context{
		Roles: make([]string, 0, len(t.Roles)),
		Auth: map[string]string{
			"user_id":             t.User.ID,
			"user_name":           t.User.Name,
			"user_domain_id":      t.User.Domain.ID,
			"user_domain_name":    t.User.Domain.Name,
			"domain_id":           t.DomainScope.ID,
			"domain_name":         t.DomainScope.Name,
			"project_id":          t.ProjectScope.ID,
			"project_name":        t.ProjectScope.Name,
			"project_domain_id":   t.ProjectScope.Domain.ID,
			"project_domain_name": t.ProjectScope.Domain.Name,
			"tenant_id":           t.ProjectScope.ID,
			"tenant_name":         t.ProjectScope.Name,
			"tenant_domain_id":    t.ProjectScope.Domain.ID,
			"tenant_domain_name":  t.ProjectScope.Domain.Name,
		},
		Request: nil,
		Logger:  util.LogDebug,
	}
	for key, value := range c.Auth {
		if value == "" {
			delete(c.Auth, key)
		}
	}
	for _, role := range t.Roles {
		c.Roles = append(c.Roles, role.Name)
	}
	if c.Request == nil {
		c.Request = map[string]string{}
	}

	return c
}

//RefreshToken fetches a new Keystone auth token. It is also used
//to fetch the initial token on startup.
func (d keystone) RefreshToken() error {
	//NOTE: This function is very similar to v3auth() in
	//gophercloud/openstack/client.go, but with a few differences:
	//
	//1. thread-safe token renewal
	//2. proper support for cross-domain scoping

	util.LogDebug("renewing Keystone token...")

	d.TokenRenewalMutex.Lock()
	defer d.TokenRenewalMutex.Unlock()

	providerClient.TokenID = ""

	//TODO: crashes with RegionName != ""
	eo := gophercloud.EndpointOpts{Region: ""}
	keystone, err := openstack.NewIdentityV3(providerClient, eo)
	if err != nil {
		return fmt.Errorf("cannot initialize Keystone client: %v", err)
	}

	util.LogDebug("Keystone URL: %s", keystone.Endpoint)

	result := tokens.Create(keystone, d.AuthOptions())
	token, err := result.ExtractToken()
	if err != nil {
		return fmt.Errorf("cannot read token: %v", err)
	}
	catalog, err := result.ExtractServiceCatalog()
	if err != nil {
		return fmt.Errorf("cannot read service catalog: %v", err)
	}

	providerClient.TokenID = token.ID
	providerClient.ReauthFunc = d.RefreshToken //TODO: exponential backoff necessary or already provided by gophercloud?
	providerClient.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		return openstack.V3EndpointURL(catalog, opts)
	}

	return nil
}

func (d keystone) AuthOptions() *gophercloud.AuthOptions {
	return &gophercloud.AuthOptions{
		IdentityEndpoint: viper.GetString("keystone.auth_url"),
		Username:         viper.GetString("keystone.username"),
		Password:         viper.GetString("keystone.password"),
		DomainName:       viper.GetString("keystone.user_domain_name"),
		// Note: gophercloud only allows for user & project in the same domain
		TenantName: viper.GetString("keystone.project_name"),
	}
}

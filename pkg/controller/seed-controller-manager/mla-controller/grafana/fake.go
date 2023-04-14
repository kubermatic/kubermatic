/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grafana

import (
	"context"
	"errors"

	grafanasdk "github.com/kubermatic/grafanasdk"
)

type fakeDatabase struct {
	Orgs        map[string]grafanasdk.Org
	OrgUsers    map[uint]map[uint]grafanasdk.OrgUser
	Users       map[string]grafanasdk.User
	Datasources map[uint]map[string]grafanasdk.Datasource
	Dashboards  map[uint]map[string]grafanasdk.Board
	DefaultOrg  uint
}

func newFakeDatabase() *fakeDatabase {
	return &fakeDatabase{
		Orgs:        map[string]grafanasdk.Org{},
		OrgUsers:    map[uint]map[uint]grafanasdk.OrgUser{},
		Users:       map[string]grafanasdk.User{},
		Datasources: map[uint]map[string]grafanasdk.Datasource{},
		Dashboards:  map[uint]map[string]grafanasdk.Board{},
	}
}

// A fake struct is *not* safe for concurrent use!
type FakeGrafana struct {
	Database   *fakeDatabase
	currentOrg uint
}

func NewFakeClient() *FakeGrafana {
	return &FakeGrafana{
		Database: newFakeDatabase(),
	}
}

var _ Client = &FakeGrafana{}

func (f *FakeGrafana) CreateDefaultOrg(org grafanasdk.Org) error {
	if len(f.Database.Orgs) > 0 {
		return errors.New("default org must be created first")
	}

	status, err := f.CreateOrg(context.Background(), org)
	if err != nil {
		return err
	}

	f.Database.DefaultOrg = *status.ID

	return nil
}

func (f *FakeGrafana) CreateOrg(_ context.Context, org grafanasdk.Org) (grafanasdk.StatusMessage, error) {
	if _, ok := f.Database.Orgs[org.Name]; ok {
		return grafanasdk.StatusMessage{}, errors.New("org with this name already exists")
	}

	org.ID = uint(len(f.Database.Orgs) + 1)
	f.Database.Orgs[org.Name] = org

	return grafanasdk.StatusMessage{ID: &org.ID}, nil
}

func (f *FakeGrafana) GetOrgByOrgName(_ context.Context, name string) (grafanasdk.Org, error) {
	org, ok := f.Database.Orgs[name]
	if !ok {
		return grafanasdk.Org{}, grafanasdk.ErrNotFound{}
	}

	return org, nil
}

func (f *FakeGrafana) WithOrgIDHeader(orgID uint) Client {
	return &FakeGrafana{
		Database:   f.Database,
		currentOrg: orgID,
	}
}

func (f *FakeGrafana) SetOrgIDHeader(orgID uint) {
	f.currentOrg = orgID
}

func (f *FakeGrafana) CreateOAuthUser(ctx context.Context, email string) (*grafanasdk.User, error) {
	if _, ok := f.Database.Users[email]; ok {
		return nil, errors.New("user with this e-mail already exists")
	}

	user := grafanasdk.User{
		ID:    uint(len(f.Database.Users) + 1),
		Email: email,
	}
	f.Database.Users[email] = user

	// add user to default org
	if f.Database.DefaultOrg > 0 {
		role := grafanasdk.UserRole{
			LoginOrEmail: email,
			Role:         string(grafanasdk.ROLE_VIEWER),
		}

		if _, err := f.AddOrgUser(ctx, role, f.Database.DefaultOrg); err != nil {
			return nil, err
		}
	}

	return &user, nil
}

func (f *FakeGrafana) LookupUser(_ context.Context, nameOrEmail string) (grafanasdk.User, error) {
	user, ok := f.Database.Users[nameOrEmail]
	if !ok {
		return grafanasdk.User{}, grafanasdk.ErrNotFound{}
	}

	return user, nil
}

func (f *FakeGrafana) GetOrgUsers(_ context.Context, orgID uint) ([]grafanasdk.OrgUser, error) {
	result := []grafanasdk.OrgUser{}

	for userID := range f.Database.OrgUsers[orgID] {
		result = append(result, f.Database.OrgUsers[orgID][userID])
	}

	return result, nil
}

func (f *FakeGrafana) GetOrgUser(ctx context.Context, orgID, userID uint) (*grafanasdk.OrgUser, error) {
	users, err := f.GetOrgUsers(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.ID == userID {
			return &user, nil
		}
	}

	return nil, nil
}

func (f *FakeGrafana) getUserByID(userID uint) *grafanasdk.User {
	for _, user := range f.Database.Users {
		if user.ID == userID {
			return &user
		}
	}

	return nil
}

func (f *FakeGrafana) AddOrgUser(ctx context.Context, userRole grafanasdk.UserRole, orgID uint) (grafanasdk.StatusMessage, error) {
	user, err := f.LookupUser(ctx, userRole.LoginOrEmail)
	if err != nil {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	if _, ok := f.Database.OrgUsers[orgID][user.ID]; ok {
		return grafanasdk.StatusMessage{}, errors.New("user is already part of this organization")
	}

	if _, ok := f.Database.OrgUsers[orgID]; !ok {
		f.Database.OrgUsers[orgID] = map[uint]grafanasdk.OrgUser{}
	}

	f.Database.OrgUsers[orgID][user.ID] = grafanasdk.OrgUser{
		ID:    user.ID,
		OrgId: orgID,
		Email: user.Email,
		Login: user.Login,
		Role:  userRole.Role,
	}

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) UpdateOrgUser(ctx context.Context, userRole grafanasdk.UserRole, orgID uint, userID uint) (grafanasdk.StatusMessage, error) {
	if _, ok := f.Database.OrgUsers[orgID]; !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	user, ok := f.Database.OrgUsers[orgID][userID]
	if !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	user.Role = userRole.Role
	f.Database.OrgUsers[orgID][userID] = user

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) DeleteOrgUser(ctx context.Context, orgID uint, userID uint) (grafanasdk.StatusMessage, error) {
	if _, ok := f.Database.OrgUsers[orgID]; !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	if _, ok := f.Database.OrgUsers[orgID][userID]; !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	delete(f.Database.OrgUsers[orgID], userID)

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) DeleteGlobalUser(ctx context.Context, userID uint) (grafanasdk.StatusMessage, error) {
	for orgID, orgData := range f.Database.OrgUsers {
		if _, ok := orgData[userID]; ok {
			delete(orgData, userID)
			f.Database.OrgUsers[orgID] = orgData
		}
	}

	user := f.getUserByID(userID)
	if user == nil {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	delete(f.Database.Users, user.Email) // after this, `user` points to junk

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) CreateDatasource(ctx context.Context, ds grafanasdk.Datasource) (grafanasdk.StatusMessage, error) {
	if f.currentOrg == 0 {
		return grafanasdk.StatusMessage{}, errors.New("no current organization set")
	}

	orgDatasources, ok := f.Database.Datasources[f.currentOrg]
	if !ok {
		orgDatasources = map[string]grafanasdk.Datasource{}
	}

	if _, ok := orgDatasources[ds.UID]; ok {
		return grafanasdk.StatusMessage{}, errors.New("datasource with this UID already exists")
	}

	ds.ID = uint(len(f.Database.Datasources[f.currentOrg]) + 1)
	orgDatasources[ds.UID] = ds
	f.Database.Datasources[f.currentOrg] = orgDatasources

	return grafanasdk.StatusMessage{
		ID: &ds.ID,
	}, nil
}

func (f *FakeGrafana) GetDatasourceByName(ctx context.Context, name string) (grafanasdk.Datasource, error) {
	if f.currentOrg == 0 {
		return grafanasdk.Datasource{}, errors.New("no current organization set")
	}

	orgDatasources, ok := f.Database.Datasources[f.currentOrg]
	if !ok {
		return grafanasdk.Datasource{}, grafanasdk.ErrNotFound{}
	}

	for _, ds := range orgDatasources {
		if ds.Name == name {
			return ds, nil
		}
	}

	return grafanasdk.Datasource{}, grafanasdk.ErrNotFound{}
}

func (f *FakeGrafana) GetDatasourceByUID(ctx context.Context, uid string) (grafanasdk.Datasource, error) {
	if f.currentOrg == 0 {
		return grafanasdk.Datasource{}, errors.New("no current organization set")
	}

	orgDatasources, ok := f.Database.Datasources[f.currentOrg]
	if !ok {
		return grafanasdk.Datasource{}, grafanasdk.ErrNotFound{}
	}

	datasource, ok := orgDatasources[uid]
	if !ok {
		return grafanasdk.Datasource{}, grafanasdk.ErrNotFound{}
	}

	return datasource, nil
}

func (f *FakeGrafana) UpdateDatasource(ctx context.Context, ds grafanasdk.Datasource) (grafanasdk.StatusMessage, error) {
	if f.currentOrg == 0 {
		return grafanasdk.StatusMessage{}, errors.New("no current organization set")
	}

	orgDatasources, ok := f.Database.Datasources[f.currentOrg]
	if !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	oldDatasource, ok := orgDatasources[ds.UID]
	if !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	ds.ID = oldDatasource.ID
	f.Database.Datasources[f.currentOrg][ds.UID] = ds

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) DeleteDatasourceByUID(ctx context.Context, uid string) (grafanasdk.StatusMessage, error) {
	if f.currentOrg == 0 {
		return grafanasdk.StatusMessage{}, errors.New("no current organization set")
	}

	orgDatasources, ok := f.Database.Datasources[f.currentOrg]
	if !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	if _, ok := orgDatasources[uid]; !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	delete(orgDatasources, uid)
	f.Database.Datasources[f.currentOrg] = orgDatasources

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) SetDashboard(_ context.Context, board grafanasdk.Board, params grafanasdk.SetDashboardParams) (grafanasdk.StatusMessage, error) {
	if f.currentOrg == 0 {
		return grafanasdk.StatusMessage{}, errors.New("no current organization set")
	}

	if !params.Overwrite {
		return grafanasdk.StatusMessage{}, errors.New("must set overwrite to true")
	}

	if _, ok := f.Database.Dashboards[f.currentOrg]; !ok {
		f.Database.Dashboards[f.currentOrg] = map[string]grafanasdk.Board{}
	}
	f.Database.Dashboards[f.currentOrg][board.UID] = board

	return grafanasdk.StatusMessage{}, nil
}

func (f *FakeGrafana) DeleteDashboardByUID(ctx context.Context, uid string) (grafanasdk.StatusMessage, error) {
	if f.currentOrg == 0 {
		return grafanasdk.StatusMessage{}, errors.New("no current organization set")
	}

	orgDashboards, ok := f.Database.Dashboards[f.currentOrg]
	if !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	if _, ok := orgDashboards[uid]; !ok {
		return grafanasdk.StatusMessage{}, grafanasdk.ErrNotFound{}
	}

	delete(orgDashboards, uid)
	f.Database.Dashboards[f.currentOrg] = orgDashboards

	return grafanasdk.StatusMessage{}, nil
}

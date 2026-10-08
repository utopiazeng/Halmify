package service

import (
	"strings"
	"testing"

	"fileserver/internal/model"

	"gorm.io/gorm"
)

func TestIsDup(t *testing.T) {
	if !isDup(fakeErr("Error 1062 duplicate entry")) {
		t.Fatalf("expected duplicate error")
	}
	if isDup(nil) {
		t.Fatalf("nil should not be duplicate")
	}
}

func TestScopedPhysicalLookupIncludesNodeID(t *testing.T) {
	svc := NewFileService(nil, nil, nil, "fs-a")
	tenantID, nodeID, md5Hex := svc.scopedPhysicalLookup(12, " ABCDEF0123456789ABCDEF0123456789 ")
	if tenantID != 12 {
		t.Fatalf("expected tenant 12, got %d", tenantID)
	}
	if nodeID != "fs-a" {
		t.Fatalf("expected node fs-a, got %s", nodeID)
	}
	if md5Hex != strings.ToLower(strings.TrimSpace(" ABCDEF0123456789ABCDEF0123456789 ")) {
		t.Fatalf("expected normalized md5, got %s", md5Hex)
	}
}

func TestNewDeleteTaskCarriesNodeID(t *testing.T) {
	svc := NewFileService(nil, nil, nil, "fs-b")
	task := svc.newDeleteTask(8, 99, "tenant_8/aa/file.bin")
	if task.NodeID != "fs-b" {
		t.Fatalf("expected delete task node fs-b, got %s", task.NodeID)
	}
	if task.TenantID != 8 || task.PhysicalID != 99 || task.FilePath != "tenant_8/aa/file.bin" || task.Status != "pending" {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestShouldRevivePhysical(t *testing.T) {
	svc := NewFileService(nil, nil, nil, "fs-a")
	if svc.shouldRevivePhysical(model.PhysicalFile{RefCount: 1, Status: "active"}) {
		t.Fatalf("active referenced physical file should not revive")
	}
	if !svc.shouldRevivePhysical(model.PhysicalFile{RefCount: 0, Status: "active"}) {
		t.Fatalf("zero ref physical file should revive")
	}
	if !svc.shouldRevivePhysical(model.PhysicalFile{RefCount: 1, Status: "pending_delete"}) {
		t.Fatalf("non-active physical file should revive")
	}
	if !svc.shouldRevivePhysical(model.PhysicalFile{Base: model.Base{DeletedAt: gorm.DeletedAt{Valid: true}}, RefCount: 1, Status: "active"}) {
		t.Fatalf("soft-deleted physical file should revive")
	}
}

func TestTenantUsageZeroValue(t *testing.T) {
	var usage TenantUsage
	if usage.UsedBytes != 0 || usage.PhysicalCount != 0 || usage.RecordCount != 0 {
		t.Fatalf("unexpected zero usage: %+v", usage)
	}
}

type fakeErr string

func (f fakeErr) Error() string { return string(f) }

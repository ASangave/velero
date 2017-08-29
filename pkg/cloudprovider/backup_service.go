/*
Copyright 2017 Heptio Inc.

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

package cloudprovider

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/errors"

	api "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/heptio/ark/pkg/generated/clientset/scheme"
)

// BackupService contains methods for working with backups in object storage.
type BackupService interface {
	BackupGetter
	// UploadBackup uploads the specified Ark backup of a set of Kubernetes API objects, whose manifests are
	// stored in the specified file, into object storage in an Ark bucket, tagged with Ark metadata. Returns
	// an error if a problem is encountered accessing the file or performing the upload via the cloud API.
	UploadBackup(bucket, name string, metadata, backup, log io.ReadSeeker) error

	// DownloadBackup downloads an Ark backup with the specified object key from object storage via the cloud API.
	// It returns the snapshot metadata and data (separately), or an error if a problem is encountered
	// downloading or reading the file from the cloud API.
	DownloadBackup(bucket, name string) (io.ReadCloser, error)

	// DeleteBackup deletes the backup content in object storage for the given api.Backup.
	DeleteBackup(bucket, backupName string) error

	// CreateBackupLogSignedURL creates a pre-signed URL that can be used to download a backup's log
	// file from object storage. The URL expires after ttl.
	CreateBackupLogSignedURL(bucket, backupName string, ttl time.Duration) (string, error)

	// CreateBackupSignedURL creates a pre-signed URL that can be used to download a backup
	// file from object storage. The URL expires after ttl.
	CreateBackupSignedURL(bucket, backupName string, ttl time.Duration) (string, error)
}

// BackupGetter knows how to list backups in object storage.
type BackupGetter interface {
	// GetAllBackups lists all the api.Backups in object storage for the given bucket.
	GetAllBackups(bucket string) ([]*api.Backup, error)
}

const (
	metadataFileFormatString = "%s/ark-backup.json"
	backupFileFormatString   = "%s/%s.tar.gz"
	logFileFormatString      = "%s/%s.log.gz"
)

func getMetadataKey(backup string) string {
	return fmt.Sprintf(metadataFileFormatString, backup)
}

func getBackupKey(backup string) string {
	return fmt.Sprintf(backupFileFormatString, backup, backup)
}

func getBackupLogKey(backup string) string {
	return fmt.Sprintf(logFileFormatString, backup, backup)
}

type backupService struct {
	objectStorage ObjectStorageAdapter
}

var _ BackupService = &backupService{}
var _ BackupGetter = &backupService{}

// NewBackupService creates a backup service using the provided object storage adapter
func NewBackupService(objectStorage ObjectStorageAdapter) BackupService {
	return &backupService{
		objectStorage: objectStorage,
	}
}

func (br *backupService) UploadBackup(bucket, backupName string, metadata, backup, log io.ReadSeeker) error {
	// upload metadata file
	metadataKey := getMetadataKey(backupName)
	if err := br.objectStorage.PutObject(bucket, metadataKey, metadata); err != nil {
		// failure to upload metadata file is a hard-stop
		return err
	}

	// upload tar file
	if err := br.objectStorage.PutObject(bucket, getBackupKey(backupName), backup); err != nil {
		// try to delete the metadata file since the data upload failed
		deleteErr := br.objectStorage.DeleteObject(bucket, metadataKey)

		return errors.NewAggregate([]error{err, deleteErr})
	}

	// uploading log file is best-effort; if it fails, we log the error but call the overall upload a
	// success
	logKey := getBackupLogKey(backupName)
	if err := br.objectStorage.PutObject(bucket, logKey, log); err != nil {
		glog.Errorf("error uploading %s/%s: %v", bucket, logKey, err)
	}

	return nil
}

func (br *backupService) DownloadBackup(bucket, backupName string) (io.ReadCloser, error) {
	return br.objectStorage.GetObject(bucket, getBackupKey(backupName))
}

func (br *backupService) GetAllBackups(bucket string) ([]*api.Backup, error) {
	prefixes, err := br.objectStorage.ListCommonPrefixes(bucket, "/")
	if err != nil {
		return nil, err
	}
	if len(prefixes) == 0 {
		return []*api.Backup{}, nil
	}

	output := make([]*api.Backup, 0, len(prefixes))

	decoder := scheme.Codecs.UniversalDecoder(api.SchemeGroupVersion)

	for _, backupDir := range prefixes {
		err := func() error {
			key := getMetadataKey(backupDir)
			res, err := br.objectStorage.GetObject(bucket, key)
			if err != nil {
				return err
			}
			defer res.Close()

			data, err := ioutil.ReadAll(res)
			if err != nil {
				return err
			}

			obj, _, err := decoder.Decode(data, nil, nil)
			if err != nil {
				return err
			}

			backup, ok := obj.(*api.Backup)
			if !ok {
				return fmt.Errorf("unexpected type for %s/%s: %T", bucket, key, obj)
			}

			output = append(output, backup)

			return nil
		}()

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

func (br *backupService) DeleteBackup(bucket, backupName string) error {
	var errs []error

	key := getBackupKey(backupName)
	glog.V(4).Infof("Trying to delete bucket=%s, key=%s", bucket, key)
	if err := br.objectStorage.DeleteObject(bucket, key); err != nil {
		errs = append(errs, err)
	}

	key = getMetadataKey(backupName)
	glog.V(4).Infof("Trying to delete bucket=%s, key=%s", bucket, key)
	if err := br.objectStorage.DeleteObject(bucket, key); err != nil {
		errs = append(errs, err)
	}

	return errors.NewAggregate(errs)
}

func (br *backupService) CreateBackupLogSignedURL(bucket, backupName string, ttl time.Duration) (string, error) {
	return br.objectStorage.CreateSignedURL(bucket, getBackupLogKey(backupName), ttl)
}

func (br *backupService) CreateBackupSignedURL(bucket, backupName string, ttl time.Duration) (string, error) {
	return br.objectStorage.CreateSignedURL(bucket, getBackupKey(backupName), ttl)
}

// cachedBackupService wraps a real backup service with a cache for getting cloud backups.
type cachedBackupService struct {
	BackupService
	cache BackupGetter
}

// NewBackupServiceWithCachedBackupGetter returns a BackupService that uses a cache for
// GetAllBackups().
func NewBackupServiceWithCachedBackupGetter(ctx context.Context, delegate BackupService, resyncPeriod time.Duration) BackupService {
	return &cachedBackupService{
		BackupService: delegate,
		cache:         NewBackupCache(ctx, delegate, resyncPeriod),
	}
}

func (c *cachedBackupService) GetAllBackups(bucketName string) ([]*api.Backup, error) {
	return c.cache.GetAllBackups(bucketName)
}

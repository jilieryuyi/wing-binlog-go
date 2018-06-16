package consul

import "github.com/hashicorp/consul/api"

type KvEntity struct {
	kv IKv
	Key string
	Value []byte
}

func NewKvEntity(kv *api.KV, key string, value []byte) *KvEntity{
	return &KvEntity{NewKv(kv), key, value}
}

func (kv *KvEntity) Set() (*KvEntity, error) {
	err := kv.kv.Set(kv.Key, kv.Value)
	return kv, err
}

func (kv *KvEntity) Get() (*KvEntity, error) {
	v, err := kv.kv.Get(kv.Key)
	kv.Value = v
	return kv, err
}

func (kv *KvEntity) Delete() (*KvEntity, error) {
	err := kv.kv.Delete(kv.Key)
	return kv, err
}
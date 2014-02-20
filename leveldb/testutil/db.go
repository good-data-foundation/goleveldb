// Copyright (c) 2014, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testutil

import (
	"fmt"
	"math/rand"

	. "github.com/onsi/gomega"

	"github.com/syndtr/goleveldb/leveldb/util"
)

type DBAct int

func (a DBAct) String() string {
	switch a {
	case DBNone:
		return "none"
	case DBPut:
		return "put"
	case DBOverwrite:
		return "overwrite"
	case DBDelete:
		return "delete"
	case DBDeleteNA:
		return "delete_na"
	}
	return "unknown"
}

const (
	DBNone DBAct = iota
	DBPut
	DBOverwrite
	DBDelete
	DBDeleteNA
)

type Put interface {
	Put(key []byte, value []byte) error
}

type Delete interface {
	Delete(key []byte) error
}

type DBTesting struct {
	Rand *rand.Rand
	DB   interface {
		Get
		Put
		Delete
	}
	PostFn             func(t *DBTesting)
	Deleted, Present   KeyValue
	Act, LastAct       DBAct
	ActKey, LastActKey []byte
}

func (t *DBTesting) post() {
	if t.PostFn != nil {
		t.PostFn(t)
	}
}

func (t *DBTesting) setAct(act DBAct, key []byte) {
	t.LastAct, t.Act = t.Act, act
	t.LastActKey, t.ActKey = t.ActKey, key
}

func (t *DBTesting) text() string {
	return fmt.Sprintf("last action was <%v> %q, <%v> %q", t.LastAct, t.LastActKey, t.Act, t.ActKey)
}

func (t *DBTesting) Text() string {
	return "DBTesting " + t.text()
}

func (t *DBTesting) TestPresentKV(key, value []byte) {
	rvalue, err := t.DB.Get(key)
	Expect(err).ShouldNot(HaveOccurred(), "Get on key %q, %s", key, t.text())
	Expect(rvalue).Should(Equal(value), "Value for key %q, %s", key, t.text())
}

func (t *DBTesting) TestAllPresent() {
	t.Present.IterateShuffled(t.Rand, func(i int, key, value []byte) {
		t.TestPresentKV(key, value)
	})
}

func (t *DBTesting) TestDeletedKey(key []byte) {
	_, err := t.DB.Get(key)
	Expect(err).Should(Equal(util.ErrNotFound), "Get on deleted key %q, %s", key, t.text())
}

func (t *DBTesting) TestAllDeleted() {
	t.Deleted.IterateShuffled(t.Rand, func(i int, key, value []byte) {
		t.TestDeletedKey(key)
	})
}

func (t *DBTesting) TestAll() {
	dn := t.Deleted.Len()
	pn := t.Present.Len()
	ShuffledIndex(t.Rand, dn+pn, 1, func(i int) {
		if i >= dn {
			key, value := t.Present.Index(i - dn)
			t.TestPresentKV(key, value)
		} else {
			t.TestDeletedKey(t.Deleted.KeyAt(i))
		}
	})
}

func (t *DBTesting) Put(key, value []byte) {
	if new := t.Present.PutU(key, value); new {
		t.setAct(DBPut, key)
	} else {
		t.setAct(DBOverwrite, key)
	}
	t.Deleted.Delete(key)
	err := t.DB.Put(key, value)
	Expect(err).ShouldNot(HaveOccurred(), t.Text())
	t.TestPresentKV(key, value)
	t.post()
}

func (t *DBTesting) PutRandom() bool {
	if t.Deleted.Len() > 0 {
		i := t.Rand.Intn(t.Deleted.Len())
		key, value := t.Deleted.Index(i)
		t.Put(key, value)
		return true
	}
	return false
}

func (t *DBTesting) Delete(key []byte) {
	if exist, value := t.Present.Delete(key); exist {
		t.setAct(DBDelete, key)
		t.Deleted.PutU(key, value)
	} else {
		t.setAct(DBDeleteNA, key)
	}
	err := t.DB.Delete(key)
	Expect(err).ShouldNot(HaveOccurred(), t.Text())
	t.TestDeletedKey(key)
	t.post()
}

func (t *DBTesting) DeleteRandom() bool {
	if t.Present.Len() > 0 {
		i := t.Rand.Intn(t.Present.Len())
		t.Delete(t.Present.KeyAt(i))
		return true
	}
	return false
}

func (t *DBTesting) RandomAct(round int) {
	for i := 0; i < round; i++ {
		if t.Rand.Int()%2 == 0 {
			t.PutRandom()
		} else {
			t.DeleteRandom()
		}
	}
}

func DoDBTesting(t *DBTesting) {
	if t.Rand == nil {
		t.Rand = NewRand()
	}

	t.DeleteRandom()
	t.PutRandom()
	t.DeleteRandom()
	t.DeleteRandom()
	for i := t.Deleted.Len() / 2; i >= 0; i-- {
		t.PutRandom()
	}
	t.RandomAct((t.Deleted.Len() + t.Present.Len()) * 10)
}

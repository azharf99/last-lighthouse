package core

import (
	"encoding/json"
	"fmt"
	"sort"
)

type (
	PlayerID       string
	LocationID     string
	CharacterID    string
	ComponentID    string
	ObjectiveID    string
	ArtifactID     string
	EventCardID    string
	LocationTypeID string
)

// Resource adalah enum berbasis integer, bukan string, supaya inventory bisa
// disimpan sebagai array berukuran tetap. Ini menghilangkan iterasi map dari
// jalur panas dan menjaga determinisme (lihat kontrak kemurnian di doc.go).
type Resource uint8

const (
	Wood Resource = iota
	Metal
	Crystal
	Food

	ResourceCount = 4
)

var resourceNames = [ResourceCount]string{"wood", "metal", "crystal", "food"}

func (r Resource) String() string {
	if int(r) >= ResourceCount {
		return "unknown"
	}
	return resourceNames[r]
}

func (r Resource) MarshalJSON() ([]byte, error) { return json.Marshal(r.String()) }

func (r *Resource) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	res, ok := ParseResource(s)
	if !ok {
		return fmt.Errorf("core: resource tidak dikenal %q", s)
	}
	*r = res
	return nil
}

func ParseResource(s string) (Resource, bool) {
	for i, n := range resourceNames {
		if n == s {
			return Resource(i), true
		}
	}
	return 0, false
}

// ResourceSet adalah jumlah tiap resource. Array berukuran tetap: urutannya
// selalu sama, sehingga hashing state dan perbandingan replay stabil.
//
// Di JSON ia tampil sebagai objek yang enak dibaca ({"wood":2,"metal":1}) supaya
// file konten mudah di-review manusia, tetapi di memori ia tetap array.
type ResourceSet [ResourceCount]int

func (rs ResourceSet) Get(r Resource) int { return rs[r] }

func (rs ResourceSet) Total() int {
	t := 0
	for _, n := range rs {
		t += n
	}
	return t
}

func (rs ResourceSet) IsEmpty() bool { return rs.Total() == 0 }

func (rs ResourceSet) Add(other ResourceSet) ResourceSet {
	for i := range rs {
		rs[i] += other[i]
	}
	return rs
}

func (rs ResourceSet) Sub(other ResourceSet) ResourceSet {
	for i := range rs {
		rs[i] -= other[i]
	}
	return rs
}

// Covers melaporkan apakah rs memenuhi seluruh kebutuhan need.
func (rs ResourceSet) Covers(need ResourceSet) bool {
	for i := range need {
		if rs[i] < need[i] {
			return false
		}
	}
	return true
}

// Missing mengembalikan kekurangan rs terhadap need (nol kalau sudah cukup).
func (rs ResourceSet) Missing(need ResourceSet) ResourceSet {
	var out ResourceSet
	for i := range need {
		if d := need[i] - rs[i]; d > 0 {
			out[i] = d
		}
	}
	return out
}

// HasNegative dipakai untuk validasi input dari luar (command dari jaringan).
func (rs ResourceSet) HasNegative() bool {
	for _, n := range rs {
		if n < 0 {
			return true
		}
	}
	return false
}

func NewResourceSet(pairs map[Resource]int) ResourceSet {
	var rs ResourceSet
	// Iterasi map di sini aman: kita menulis ke indeks tetap, bukan
	// mengakumulasi dengan urutan yang berpengaruh.
	for r, n := range pairs {
		rs[r] = n
	}
	return rs
}

func (rs ResourceSet) MarshalJSON() ([]byte, error) {
	m := make(map[string]int, ResourceCount)
	for i, n := range rs {
		if n != 0 {
			m[resourceNames[i]] = n
		}
	}
	return json.Marshal(m)
}

func (rs *ResourceSet) UnmarshalJSON(b []byte) error {
	var m map[string]int
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	var out ResourceSet
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys) // urutan pesan error stabil
	for _, k := range keys {
		r, ok := ParseResource(k)
		if !ok {
			return fmt.Errorf("core: resource tidak dikenal %q di resource set", k)
		}
		out[r] = m[k]
	}
	*rs = out
	return nil
}

// Status adalah siklus hidup satu match.
type Status string

const (
	StatusLobby  Status = "lobby"
	StatusActive Status = "active"
	StatusWon    Status = "won"  // mercusuar menyala (GDD 6.2)
	StatusLost   Status = "lost" // Darkness mencapai 8 (GDD 6.1)
)

// Phase adalah fase dalam satu ronde (GDD 12).
type Phase string

const (
	PhaseEvent    Phase = "event"    // GDD 13
	PhasePlayer   Phase = "player"   // GDD 14
	PhaseMonster  Phase = "monster"  // GDD 15
	PhaseDarkness Phase = "darkness" // GDD 22
)

package memdb

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/moson-mo/goaurrpc/internal/aur"
)

// LoadDbFromFile loads package data from local JSON file
func LoadDbFromFile(path string) (*MemoryDB, error) {
	var b []byte
	if strings.HasSuffix(path, ".gz") {
		gz, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r, err := gzip.NewReader(gz)
		if err != nil {
			return nil, err
		}
		b, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		b, err = ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}

	return bytesToMemoryDB(b)
}

// LoadDbFromUrl loads package data from web hosted file (packages-meta-ext-v1.json.gz)
func LoadDbFromUrl(url string, lastmod string) (*MemoryDB, string, error) {
	b, lastmod, err := aur.DownloadPackageData(url, lastmod)
	if err != nil {
		return nil, "", err
	}
	memdb, err := bytesToMemoryDB(b)
	if err != nil {
		return nil, "", err
	}
	return memdb, lastmod, nil
}

// constructs MemoryDB struct
func bytesToMemoryDB(b []byte) (*MemoryDB, error) {
	db := MemoryDB{}
	err := json.Unmarshal(b, &db.PackageSlice)
	if err != nil {
		return nil, err
	}

	db.fillHelperVars()

	return &db, nil
}

// fills some slices we need for search lookups.
func (db *MemoryDB) fillHelperVars() {
	n := len(db.PackageSlice)

	db.PackageMap = make(map[string]PackageInfo, n)
	db.PackageNames = make([]string, 0, n)
	db.PackageDescriptions = make([]PackageDescription, 0, n)
	db.References = map[string][]*PackageInfo{}
	baseNames := []string{}

	for i, pkg := range db.PackageSlice {
		db.PackageMap[pkg.Name] = pkg
		db.PackageNames = append(db.PackageNames, pkg.Name)
		baseNames = append(baseNames, pkg.PackageBase)
		db.PackageDescriptions = append(db.PackageDescriptions, PackageDescription{Name: pkg.Name, Description: pkg.Description})

		// depends
		for _, ref := range pkg.Depends {
			sref := "dep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// makedepends
		for _, ref := range pkg.MakeDepends {
			sref := "mdep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// optdepends
		for _, ref := range pkg.OptDepends {
			sref := "odep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// checkdepends
		for _, ref := range pkg.CheckDepends {
			sref := "cdep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// provides
		for _, ref := range pkg.Provides {
			sref := "pro-" + stripRef(ref)
			if ref != pkg.Name {
				db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
			}
		}
		// conflicts
		for _, ref := range pkg.Conflicts {
			sref := "con-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// replaces
		for _, ref := range pkg.Replaces {
			sref := "rep-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// groups
		for _, ref := range pkg.Groups {
			sref := "grp-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
		// keywords
		for _, ref := range pkg.Keywords {
			sref := "key-" + stripRef(ref)
			db.References[sref] = append(db.References[sref], &db.PackageSlice[i])
		}
	}

	db.PackageBaseNames = distinctStringSlice(baseNames)

	sort.Strings(db.PackageBaseNames)
	sort.Strings(db.PackageNames)
}

func stripRef(ref string) string {
	ret := strings.Split(ref, ">")[0]
	ret = strings.Split(ret, "<")[0]
	ret = strings.Split(ret, ":")[0]
	ret = strings.Split(ret, "=")[0]
	return ret
}

func distinctStringSlice(s []string) []string {
	keys := make(map[string]bool)
	dist := []string{}

	for _, entry := range s {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			dist = append(dist, entry)
		}
	}
	return dist
}

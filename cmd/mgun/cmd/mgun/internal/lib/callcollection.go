package lib

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

var (
	arrayParamRegexp  = regexp.MustCompile(`[\w\d\-\_]\[\]+`)
	configParamRegexp = regexp.MustCompile(`\$\{([\w\d\-\_\.]+)\}`)
	callCollection    = &CallCollection{
		Features:   make(Features, 0),
		Calibers:   make(CaliberMap),
		Cartridges: make(Cartridges, 0),
	}
)

// CallCollection collection of parameters for a call
type CallCollection struct {
	Features   Features   `yaml:"headers"`
	Calibers   CaliberMap `yaml:"params"`
	Cartridges Cartridges `yaml:"requests"`
}

// GetCallCollection collection of definitions of hits to be made
func GetCallCollection() *CallCollection {
	return callCollection
}

func (cc *CallCollection) prepare() {
	if len(cc.Cartridges) == 0 {
		cartridge := new(Cartridge)
		cartridge.path = NewNamedDescribedFeature(GET_METHOD, "/")
		cc.Cartridges = append(cc.Cartridges, cartridge)
	}
	reporter.log("cartridges count - %v", cc.Cartridges)
}

// findCaliber check a call
func (cc *CallCollection) findCaliber(path string) *Caliber {
	parts := strings.Split(path, ".")
	if caliber, ok := cc.Calibers[parts[0]]; ok {
		return cc.findInCaliber(caliber, parts[1:])
	}
	return nil
}

// findInCaliber get a random call from the list in the configuration
func (cc *CallCollection) findInCaliber(caliber *Caliber, pathParts []string) *Caliber {
	nextPathParts := cc.getNextPathParts(pathParts)
	switch caliber.kind {
	case CALIBER_KIND_LIST:
		calibers := caliber.feature.description.(CaliberList)
		rand.Seed(time.Now().UnixNano())
		randCaliber := calibers[rand.Intn(len(calibers))]
		if randCaliber.kind == CALIBER_KIND_MAP {
			return cc.findInCaliber(randCaliber, nextPathParts)
		} else {
			return randCaliber
		}
	case CALIBER_KIND_MAP:
		caliberMap := caliber.feature.description.(CaliberMap)
		if childCaliber, ok := caliberMap[pathParts[0]]; ok {
			return cc.findInCaliber(childCaliber, nextPathParts)
		} else {
			return nil
		}
	case CALIBER_KIND_SESSION:
		return caliber
	case CALIBER_KIND_SIMPLES:
		return nil
	default:
		return caliber
	}
}

func (cc *CallCollection) getNextPathParts(pathParts []string) []string {
	if len(pathParts) > 1 {
		return pathParts[1:]
	} else {
		return pathParts
	}
}

type CaliberList []*Caliber
type CaliberMap map[string]*Caliber
type CaliberMapList []CaliberMap

func (cm CaliberMap) UnmarshalYAML(unmarshal func(yaml interface{}) error) error {
	reporter.log("unmarshal calibers")

	calibers := make(map[interface{}]interface{})
	err := unmarshal(calibers)

	if rawSessions, ok := calibers["session"].([]interface{}); ok {
		reporter.log("fill session calibers")

		delete(calibers, "session")
		list := make(CaliberList, 0)

		for _, rawSession := range rawSessions {
			if session, ok := rawSession.(map[interface{}]interface{}); ok {
				reporter.ln()
				caliberMap := make(CaliberMap)
				caliberMap.fill(session)
				list = append(list, NewCaliberByKindAndFeature(CALIBER_KIND_MAP, NewDescribedFeature(caliberMap)))
			}
		}
		cm["session"] = NewCaliberByKindAndFeature(CALIBER_KIND_SESSION, NewDescribedFeature(list))
	}

	reporter.ln()
	reporter.log("fill other calibers")
	cm.fill(calibers)
	return err
}

func (cm CaliberMap) fill(rawCalibers map[interface{}]interface{}) {
	for rawKey, rawValue := range rawCalibers {
		var caliber *Caliber
		key := fmt.Sprintf("%v", rawKey)

		switch rawValue.(type) {
		case []interface{}:
			if arrayParamRegexp.MatchString(key) {
				caliber = NewCaliberByKindAndFeature(CALIBER_KIND_SIMPLES, NewDescribedFeature(rawValue))
			} else {
				list := make(CaliberList, 0)
				list.fill(rawValue.([]interface{}))
				caliber = NewCaliberByKindAndFeature(CALIBER_KIND_LIST, NewDescribedFeature(list))
			}
			break
		case map[interface{}]interface{}:
			caliberMap := make(CaliberMap)
			caliberMap.fill(rawValue.(map[interface{}]interface{}))
			caliber = NewCaliberByKindAndFeature(CALIBER_KIND_MAP, NewDescribedFeature(caliberMap))
			break
		default:
			caliber = NewCaliberByKindAndFeature(CALIBER_KIND_SIMPLE, NewDescribedFeature(rawValue))
			break
		}
		cm[key] = caliber
		reporter.log("caliber: key - %v,  kind - %v, feature - %v, children - %v", key, caliber.kind, caliber.feature, caliber.children)
	}
}

func (cl *CaliberList) fill(rawList []interface{}) {
	for _, value := range rawList {
		switch value.(type) {
		case []interface{}:
			list := make(CaliberList, 0)
			list.fill(value.([]interface{}))
			*cl = append(*cl, NewCaliberByKindAndFeature(CALIBER_KIND_LIST, NewDescribedFeature(list)))
			break
		case map[interface{}]interface{}:
			caliberMap := make(CaliberMap)
			caliberMap.fill(value.(map[interface{}]interface{}))
			*cl = append(*cl, NewCaliberByKindAndFeature(CALIBER_KIND_MAP, NewDescribedFeature(caliberMap)))
			break
		default:
			*cl = append(*cl, NewCaliberByKindAndFeature(CALIBER_KIND_SIMPLE, NewDescribedFeature(value)))
			break
		}
	}
}

type Caliber struct {
	kind     CaliberKind
	feature  *Feature
	children interface{}
}

func NewCaliber() *Caliber {
	return new(Caliber)
}

func NewCaliberByKind(kind CaliberKind) *Caliber {
	caliber := NewCaliber()
	caliber.kind = kind
	return caliber
}

func NewCaliberByKindAndFeature(kind CaliberKind, feature *Feature) *Caliber {
	caliber := NewCaliberByKind(kind)
	caliber.feature = feature
	return caliber
}

type CaliberKind int

const (
	CALIBER_KIND_SIMPLE CaliberKind = iota
	CALIBER_KIND_SIMPLES
	CALIBER_KIND_MAP
	CALIBER_KIND_LIST
	CALIBER_KIND_SESSION
)

type Cartridges []*Cartridge

func (c *Cartridges) UnmarshalYAML(unmarshal func(yaml interface{}) error) error {
	reporter.ln()
	reporter.log("unmarchal cartridges")

	rawCartridges := make([]interface{}, 0)
	err := unmarshal(&rawCartridges)

	c.fill(rawCartridges)

	return err
}

func (c *Cartridges) fill(rawCartridges []interface{}) {
	for _, rawCartridge := range rawCartridges {
		cartridge := &Cartridge{
			successStatusCodes: []int{200, 301, 302},
		}
		for rawKey, rawValue := range rawCartridge.(map[interface{}]interface{}) {
			key := rawKey.(string)
			switch key {
			case GET_METHOD, POST_METHOD, PUT_METHOD, DELETE_METHOD:
				kill.shotsCount++
				cartridge.id = kill.shotsCount
				cartridge.path = NewNamedDescribedFeature(key, rawValue)
				cartridge.path.rawDescription = rawValue
				break
			case RANDOM_METHOD, SYNC_METHOD:
				cartridge.path = NewNamedFeature(key)
				cartridge.children = make(Cartridges, 0)
				cartridge.children.fill(rawValue.([]interface{}))
				break
			case "headers":
				cartridge.bulletFeatures = make(Features, 0)
				cartridge.bulletFeatures.fill(rawValue.(map[interface{}]interface{}))
				break
			case "params":
				cartridge.chargeFeatures = make(Features, 0)
				cartridge.chargeFeatures.fill(rawValue.(map[interface{}]interface{}))
				break
			case "timeout":
				cartridge.timeout = time.Duration(rawValue.(int))
				break
				//			case "successStatusCodes":
				//				cartridge.timeout = time.Duration(rawValue.(int))
				//				break;
				//			case "failedStatusCodes":
				//				cartridge.timeout = time.Duration(rawValue.(int))
				//				break;
			}
		}
		*c = append(*c, cartridge)
		reporter.log(
			"cartridge: path - %v,  bulletFeatures - %v, chargeFeatures - %v, timeout - %v, children - %v",
			cartridge.path,
			cartridge.bulletFeatures,
			cartridge.chargeFeatures,
			cartridge.timeout,
			cartridge.children,
		)
	}
}

func (c *Cartridges) getCodes(rawCodes interface{}) []int {
	switch rawCodes.(type) {
	case []interface{}:
		codes := make([]int, 0)
		for _, rawCode := range rawCodes.([]interface{}) {
			codes = append(codes, rawCode.(int))
		}
		return codes
	default:
		return []int{rawCodes.(int)}
	}
}

func (c Cartridges) toPlainSlice() Cartridges {
	cartridges := make(Cartridges, 0)
	for _, cartridge := range c {
		if cartridge.path.name == RANDOM_METHOD || cartridge.path.name == SYNC_METHOD {
			cartridges = append(cartridges, cartridge.children.toPlainSlice()...)
		} else {
			cartridges = append(cartridges, cartridge)
		}
	}
	return cartridges
}

const (
	GET_METHOD     = "GET"
	POST_METHOD    = "POST"
	PUT_METHOD     = "PUT"
	DELETE_METHOD  = "DELETE"
	RANDOM_METHOD  = "RANDOM"
	SYNC_METHOD    = "SYNC"
	INCLUDE_METHOD = "INCLUDE"
)

type Cartridge struct {
	id                 int
	path               *Feature
	bulletFeatures     Features
	chargeFeatures     Features
	timeout            time.Duration
	successStatusCodes []int
	failedStatusCodes  []int
	children           Cartridges
}

func (c *Cartridge) getMethod() string {
	return c.path.name
}

func (c *Cartridge) getPathAsString(killer *Killer) string {
	return c.path.String(killer)
}

func (c *Cartridge) getChildren() Cartridges {
	if c.path.name == RANDOM_METHOD {
		shuffleChildren := make(Cartridges, len(c.children))
		rand.Seed(time.Now().UnixNano())
		indexes := rand.Perm(len(c.children))
		for i, v := range indexes {
			shuffleChildren[v] = c.children[i]
		}
		return shuffleChildren
	} else if c.path.name == SYNC_METHOD {
		return c.children
	} else {
		return Cartridges{}
	}
}

type FeatureKind int

const (
	FEATURE_KIND_SIMPLE FeatureKind = iota
	FEATURE_KIND_MULTIPLE
)

type Features []*Feature

func (f *Features) UnmarshalYAML(unmarshal func(yaml interface{}) error) error {
	rawFeatures := make(map[interface{}]interface{})
	err := unmarshal(&rawFeatures)

	f.fill(rawFeatures)

	return err
}

func (f *Features) fill(rawFeatures map[interface{}]interface{}) {
	for rawKey, rawValue := range rawFeatures {
		key := rawKey.(string)
		value := fmt.Sprintf("%v", rawValue)
		*f = append(*f, NewNamedDescribedFeature(key, value))
	}
}

type Feature struct {
	name           string
	description    interface{}
	rawDescription interface{}
	units          []string
	kind           FeatureKind
}

func NewFeature() *Feature {
	return new(Feature)
}

func NewNamedFeature(name string) *Feature {
	feature := NewFeature()
	feature.name = name
	return feature
}

func NewDescribedFeature(rawDescription interface{}) *Feature {
	return NewFeature().setDescription(rawDescription)
}

func NewNamedDescribedFeature(name string, rawDescription interface{}) *Feature {
	return NewNamedFeature(name).setDescription(rawDescription)
}

func (f *Feature) setDescription(rawDescription interface{}) *Feature {
	switch rawDescription.(type) {
	case string:
		description := rawDescription.(string)
		if configParamRegexp.MatchString(description) {
			f.description = configParamRegexp.ReplaceAllString(description, "%v")
			f.units = make([]string, 0)
			for _, submatches := range configParamRegexp.FindAllStringSubmatch(description, -1) {
				if len(submatches) >= 2 {
					f.units = append(f.units, submatches[1])
				}
			}
			f.kind = FEATURE_KIND_MULTIPLE
		} else {
			f.setSimpleDescription(description)
		}
		break
	default:
		f.setSimpleDescription(rawDescription)
		break
	}
	return f
}

func (f *Feature) setSimpleDescription(description interface{}) *Feature {
	f.description = description
	f.kind = FEATURE_KIND_SIMPLE
	return f
}

func (f *Feature) String(killer *Killer) string {
	if f.kind == FEATURE_KIND_SIMPLE {
		return fmt.Sprintf("%v", f.description)
	}
	values := make([]interface{}, len(f.units))
	for i, unit := range f.units {
		reporter.log("find caliber by unit - %v", unit)
		caliber := callCollection.findCaliber(unit)
		if caliber != nil && caliber.kind == CALIBER_KIND_SESSION {
			if killer.session == nil {
				calibers := caliber.feature.description.(CaliberList)
				rand.Seed(time.Now().UnixNano())
				killer.session = calibers[rand.Intn(len(calibers))]
			}
			caliber = callCollection.findInCaliber(
				killer.session,
				callCollection.getNextPathParts(strings.Split(unit, ".")),
			)
		}
		if caliber != nil {
			values[i] = caliber.feature.String(killer)
		}
	}
	return fmt.Sprintf(f.description.(string), values...)
}

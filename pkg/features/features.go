package features

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// FeatureGate is feature gate 'manager' to be used
var FeatureGate *featureGate

type maturity string

const (
	// Alpha - alpha version
	Alpha = maturity("ALPHA")
	// Beta - beta version
	Beta = maturity("BETA")
	// GA - general availability
	GA = maturity("GA")

	// Deprecated - feature that will be deprecated in 2 releases
	Deprecated = maturity("DEPRECATED")
)

// Feature name
type feature string

// List of supported features
const (
	// AlphaFeature - description
	AlphaFeature feature = "alphaFeature"

	// BetaFeature - description
	BetaFeature feature = "betaFeature"

	// GaFeature - description
	GaFeature feature = "gaFeature"

	// DeprecatedFeature - description
	DeprecatedFeature feature = "deprecatedFeature"
)

type featureSpec struct {
	// Default is the default enablement state for the feature
	Default bool
	// Maturity indicates the maturity level of the feature
	Maturity maturity
}

var defaultSriovDpFeatureGates = map[feature]featureSpec{
	AlphaFeature:      {Default: false, Maturity: Alpha},
	BetaFeature:       {Default: true, Maturity: Beta},
	GaFeature:         {Default: true, Maturity: GA},
	DeprecatedFeature: {Default: false, Maturity: Deprecated},
}

type featureGate struct {
	knownFeatures map[feature]featureSpec
	enabled       map[feature]bool
}

func init() {
	FeatureGate = newFeatureGate()
}

func newFeatureGate() *featureGate {
	if FeatureGate != nil {
		return FeatureGate
	}
	fg := &featureGate{}
	fg.knownFeatures = make(map[feature]featureSpec)
	fg.enabled = make(map[feature]bool)

	for k, v := range defaultSriovDpFeatureGates {
		fg.knownFeatures[k] = v
	}

	for k, v := range fg.knownFeatures {
		fg.enabled[k] = v.Default
	}
	return fg
}

// Enabled returns enabelement status of the provided feature
func (fg *featureGate) Enabled(f feature) bool {
	return FeatureGate.enabled[f]
}

func (fg *featureGate) isFeatureSupported(f feature) bool {
	_, exists := fg.knownFeatures[f]
	return exists
}

func (fg *featureGate) set(f feature, status bool) error {
	if !fg.isFeatureSupported(f) {
		return fmt.Errorf("Feature %s is not supported", f)
	}
	fg.enabled[f] = status
	if status == true && fg.knownFeatures[f].Maturity == Deprecated {
		glog.Warningf("WARNING: Feature %s will be deprecated soon", f)
	}
	return nil
}

// SetFromMap sets the enablement status of featuers accordig to a map
func (fg *featureGate) SetFromMap(valuesToSet map[string]bool) error {
	for k, v := range valuesToSet {
		err := fg.set(feature(k), v)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetFromString converts config string to map and sets the enablement status of the selected features
// copied form k8s and slightly changed - TBC?
func (fg *featureGate) SetFromString(value string) error {
	featureMap := make(map[string]bool)
	for _, s := range strings.Split(value, ",") {
		if len(s) == 0 {
			continue
		}
		splitted := strings.Split(s, "=")
		key := strings.TrimSpace(splitted[0])
		if len(splitted) != 2 {
			if len(splitted) > 2 {
				return fmt.Errorf("too many values for %s", key)
			}
			return fmt.Errorf("enablement value for %s is missing", key)
		}

		val := strings.TrimSpace(splitted[1])
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error while processing %s=%s, err: %v", key, val, err)
		}

		featureMap[key] = boolVal
	}
	return fg.SetFromMap(featureMap)
}

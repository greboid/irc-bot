package bot

import (
	"reflect"
	"testing"
)

func Test_getJoinCommands(t *testing.T) {
	bot := Bot{}
	tests := []struct {
		name             string
		args             string
		wantJoinCommands []string
	}{
		{
			name:             "single channel",
			args:             "#test",
			wantJoinCommands: []string{"#test"},
		},
		{
			name:             "multiple channel",
			args:             "#test,#test2",
			wantJoinCommands: []string{"#test,#test2"},
		},
		{
			name:             "single keyed channel",
			args:             "#test key",
			wantJoinCommands: []string{"#test key"},
		},
		{
			name:             "multiple keyed channel",
			args:             "#test key,#test2 key2",
			wantJoinCommands: []string{"#test,#test2 key,key2"},
		},
		{
			name: "mixed keyed/keyless channel",
			args: "#test key,#test2",
			wantJoinCommands: []string{
				"#test key",
				"#test2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotJoinCommands := bot.getJoinCommands(tt.args); !reflect.DeepEqual(gotJoinCommands, tt.wantJoinCommands) {
				t.Errorf("getJoinCommands() = %v, want %v", gotJoinCommands, tt.wantJoinCommands)
			}
		})
	}
}

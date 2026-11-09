package pdf

import (
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func TestProcessorFormDataStages(t *testing.T) {
	for _, tc := range []struct {
		name          string
		disableStages bool
		stages        []string
		wantStages    int
		wantStatus    int
	}{
		{name: "stages allowed", stages: []string{`[{"op":"render"}]`}, wantStages: 1},
		{name: "stages disabled, field absent", disableStages: true},
		{name: "stages disabled, field blank", disableStages: true, stages: []string{" "}},
		{name: "stages disabled, field set", disableStages: true, stages: []string{`[{"op":"render"}]`}, wantStatus: http.StatusBadRequest},
		{name: "stages disabled, empty list", disableStages: true, stages: []string{`[]`}, wantStatus: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &api.ContextMock{Context: new(api.Context)}
			if tc.stages != nil {
				ctx.SetValues(map[string][]string{StagesFormField: tc.stages})
			}

			p := &Processor{disableStages: tc.disableStages}
			stages, _, err := p.FormDataStages(ctx.FormData())

			if tc.wantStatus != 0 {
				status, message := api.ParseError(err)
				if status != tc.wantStatus {
					t.Fatalf("got %d %q, want %d", status, message, tc.wantStatus)
				}

				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(stages) != tc.wantStages {
				t.Errorf("got %d stages, want %d", len(stages), tc.wantStages)
			}
		})
	}
}

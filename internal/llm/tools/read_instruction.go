package tools

import "github.com/alexpls/untils/internal/models"

var readInstructionTool = tool[models.ReadInstructionParams]{
	name:        models.LLMToolNameReadInstruction,
	description: "Retrieve the instructions for the given name",
	usageBody: `- There are preconfigured instructions for common results that you might find. These provide
  templates for how to format the response, schema, and fields. Using these brings consistency
  to your responses for similar subjects.
- If it's clear from the subject that a certain instruction will apply to it, read the instruction
  before proceeding with the search. This will ensure you don't waste the user's time searching for
  things that you won't use.
- An index of all instructions will be provided to you. If one of the instructions applies to
  the result you've found, read it.
- Once you've read the instruction you may find that it doesn't actually apply to your result.
  In that case, just ignore it.
- The instructions are only a guideline and may not apply in the context of the check that
  you are peforming. In ambiguous cases default to not using the instruction.`,
	execute: func(tc *Context, params models.ReadInstructionParams) (string, error) {
		return tc.ReadInstruction(params.Name)
	},
	validate: func(tc *Context, params models.ReadInstructionParams) string {
		return noDuplicateCallsValidator(tc, params, "reading the same instruction more than once is not allowed")
	},
}

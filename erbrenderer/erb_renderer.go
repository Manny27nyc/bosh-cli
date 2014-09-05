package erbrenderer

import (
	"encoding/json"
	"path/filepath"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
)

const erbRendererLogTag = "ERBRenderer"

type ERBRenderer interface {
	Render(srcPath, dstPath string, context TemplateEvaluationContext) error
}

type erbRenderer struct {
	fs     boshsys.FileSystem
	runner boshsys.CmdRunner
	logger boshlog.Logger

	rendererScript string
}

func NewERBRenderer(
	fs boshsys.FileSystem,
	runner boshsys.CmdRunner,
	logger boshlog.Logger,
) erbRenderer {
	return erbRenderer{
		fs:     fs,
		runner: runner,
		logger: logger,

		rendererScript: templateEvaluationContextRb,
	}
}

func (r erbRenderer) Render(srcPath, dstPath string, context TemplateEvaluationContext) error {
	r.logger.Debug(erbRendererLogTag, "Rendering template %s", dstPath)

	tmpDir, err := r.fs.TempDir("erb-renderer")
	if err != nil {
		return bosherr.WrapError(err, "Creating temporary directory")
	}
	defer r.fs.RemoveAll(tmpDir)

	rendererScriptPath := filepath.Join(tmpDir, "erb-render.rb")
	err = r.writeRendererScript(rendererScriptPath)
	if err != nil {
		return err
	}

	contextPath := filepath.Join(tmpDir, "erb-context.json")
	err = r.writeContext(contextPath, context)
	if err != nil {
		return err
	}

	command := boshsys.Command{
		Name: "ruby",
		Args: []string{rendererScriptPath, contextPath, srcPath, dstPath},
	}

	_, _, _, err = r.runner.RunComplexCommand(command)
	if err != nil {
		return bosherr.WrapError(err, "Running ruby to render templates")
	}

	return nil
}

func (r erbRenderer) writeRendererScript(scriptPath string) error {
	err := r.fs.WriteFileString(scriptPath, r.rendererScript)
	if err != nil {
		return bosherr.WrapError(err, "Writing renderer script")
	}

	return nil
}

func (r erbRenderer) writeContext(contextPath string, context TemplateEvaluationContext) error {
	contextBytes, err := json.Marshal(context)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling context")
	}

	err = r.fs.WriteFileString(contextPath, string(contextBytes))
	if err != nil {
		return bosherr.WrapError(err, "Writing context")
	}

	return nil
}

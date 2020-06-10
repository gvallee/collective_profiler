all: tool_dependencies

tool_dependencies: tool_dependencies.dot
	@dot -Tpng tool_dependencies.dot -o tool_dependencies.png

clean:
	@rm -f  tool_dependencies.png

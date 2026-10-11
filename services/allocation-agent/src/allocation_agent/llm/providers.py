"""Built-in LLM providers. Importing this module registers each one.

--- Add a provider here -----------------------------------------------------------------------
Each adapter module calls register("<name>", factory) when it is imported. To add a provider
(Anthropic, OpenAI, ...):

1. write llm/<name>.py with a class that satisfies `LLMClient`, maps its errors to the
   `LLMError` subclasses, reads its own API key variable, and calls `register` at the bottom;
2. import it below, on its own line;
3. set `llm.provider: <name>` in config.yaml.

Nothing else changes: the extractor depends only on the port.
-----------------------------------------------------------------------------------------------
"""

from allocation_agent.llm import deepseek  # noqa: F401

# from allocation_agent.llm import anthropic  # future provider
# from allocation_agent.llm import openai  # future provider

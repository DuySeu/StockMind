import tiktoken

class FixedSizeChunking:
    """
    Fixed Size Chunking divides text into chunks of a fixed number of tokens. It can include an optional overlap between chunks to maintain context.
    """

    def __init__(self, chunk_size: int = 800, overlap: int = 0):
        """
        Initialize the chunking strategy with a fixed size.
        """
        if chunk_size <= 0:
            raise ValueError("chunk_size must be > 0")
        if overlap < 0:
            raise ValueError("overlap must be >= 0")
        if overlap >= chunk_size:
            raise ValueError("overlap must be < chunk_size")
        self.chunk_size = chunk_size
        self.overlap = overlap
    
    def chunk(self, text: str) -> list[str]:
        """
        Character-based fixed-width slicing of the input (simple baseline).
        - If is_tokenized=True, 'text' is assumed to be a sequence already and is sliced directly.
        """
        step = self.chunk_size - self.overlap
        if step <= 0:
            raise ValueError("chunk_size must be greater than overlap")
        return [text[i:i + self.chunk_size] for i in range(0, len(text), step)]

    def chunk_by_tokens(self, text: str, encoding_name: str = "cl100k_base") -> list[str]:
        """
        Token-based fixed-size chunking using tiktoken. Overlap is applied in tokens.
        """
        enc = tiktoken.get_encoding(encoding_name)
        tokens = enc.encode(text)
        step = self.chunk_size - self.overlap
        if step <= 0:
            raise ValueError("chunk_size must be greater than overlap")

        chunks = []
        for i in range(0, len(tokens), step):
            window = tokens[i:i + self.chunk_size]
            chunks.append(enc.decode(window))
        return chunks


class RecursiveChunking:
    """
    Recursive Chunking splits the text into smaller chunks iteratively, using a hierarchical approach with different separators or criteria.
    Initial splits are made using larger chunks, which are then further divided if necessary, aiming to keep chunks similar in size
    """

    def __init__(self, chunk_size: int = 800, overlap: int = 0):
        """
        Initialize the chunking strategy with a fixed size.
        """
        self.chunk_size = chunk_size
        self.overlap = overlap

    def chunk_by_separator(
        self,
        text: str,
        separator: str = "\n\n",
        encoding_name: str | None = "cl100k_base",
    ) -> list[str]:
        """
        Greedy packer that splits by a separator first, then packs units into chunks
        up to 'chunk_size'. If 'encoding_name' is not None, size is measured in tokens;
        otherwise it's measured in characters. Overlap is applied in the same unit.
        """
        units_raw = text.split(separator)
        if not units_raw:
            return []
        # Re-attach separators so we don't lose them between units
        units = []
        for idx, u in enumerate(units_raw):
            if idx < len(units_raw) - 1:
                units.append(u + separator)
            else:
                units.append(u)

        use_tokens = encoding_name is not None
        enc = tiktoken.get_encoding(encoding_name) if use_tokens else None

        def measure(s: str) -> int:
            if use_tokens:
                return len(enc.encode(s))
            return len(s)

        chunks: list[str] = []
        current: str = ""
        current_size = 0

        for u in units:
            u_size = measure(u)
            if current_size == 0:
                # start new
                current = u
                current_size = u_size
                # handle very large single unit by splitting it directly
                while current_size > self.chunk_size:
                    if use_tokens:
                        t = enc.encode(current)
                        chunks.append(enc.decode(t[:self.chunk_size]))
                        # start next with overlap
                        if self.overlap > 0:
                            carry = t[self.chunk_size - self.overlap:self.chunk_size]
                            current = enc.decode(carry) + enc.decode(t[self.chunk_size:])
                        else:
                            current = enc.decode(t[self.chunk_size:])
                        current_size = measure(current)
                    else:
                        chunks.append(current[:self.chunk_size])
                        if self.overlap > 0:
                            current = current[self.chunk_size - self.overlap:]
                        else:
                            current = current[self.chunk_size:]
                        current_size = measure(current)
                continue

            # try to add next unit
            if current_size + u_size <= self.chunk_size:
                current += u
                current_size += u_size
            else:
                # finalize current
                chunks.append(current)
                # prepare next with overlap
                if self.overlap > 0:
                    if use_tokens:
                        t = enc.encode(current)
                        carry = t[-self.overlap:] if self.overlap <= len(t) else t[:]
                        current = enc.decode(carry) + u
                    else:
                        carry = current[-self.overlap:] if self.overlap <= len(current) else current
                        current = carry + u
                else:
                    current = u
                current_size = measure(current)
                # If still too large (e.g., huge 'u'), split iteratively
                while current_size > self.chunk_size:
                    if use_tokens:
                        t = enc.encode(current)
                        chunks.append(enc.decode(t[:self.chunk_size]))
                        if self.overlap > 0:
                            carry = t[self.chunk_size - self.overlap:self.chunk_size]
                            current = enc.decode(carry) + enc.decode(t[self.chunk_size:])
                        else:
                            current = enc.decode(t[self.chunk_size:])
                        current_size = measure(current)
                    else:
                        chunks.append(current[:self.chunk_size])
                        if self.overlap > 0:
                            current = current[self.chunk_size - self.overlap:]
                        else:
                            current = current[self.chunk_size:]
                        current_size = measure(current)

        if current_size > 0:
            chunks.append(current)
        return chunks

class SemanticChunking:
    """
    Semantic Chunking divides the text into meaningful, semantically complete chunks based on the relationships within the text.
    Each chunk represents a complete idea or topic, maintaining the integrity of information for more accurate retrieval and generation.
    """

class AgenticChunking:
    """
    Agentic Chunking is an experimental approach that processes documents in a human-like manner.
    Chunks are created based on logical, human-like decisions about content organization, starting from the beginning and proceeding sequentially, deciding chunk boundaries dynamically.
    """

if __name__ == "__main__":
    # Simple tests you can run directly: python llm_with_python/chunking_strategies.py
    with open("./pride_and_prejudice.txt", 'r', encoding='utf-8') as file:
            document = file.read()

    # print("== Character-based fixed-width ==")
    # fchar = FixedSizeChunking(chunk_size=800, overlap=5)
    # for i, c in enumerate(fchar.chunk(document)):
    #     print(f"[{i}] ({len(c)} chars) ->", repr(c))

    print("\n== Token-based fixed-width ==")
    ftoken = FixedSizeChunking(chunk_size=200, overlap=5)
    for i, c in enumerate(ftoken.chunk_by_tokens(document)):
        print(f"[{i}] (~{len(tiktoken.get_encoding('cl100k_base').encode(c))} toks) ->", repr(c[:60]))

    # print("\n== Separator packing by tokens (\\n\\n) ==")
    # fsep_tok = FixedSizeChunking(chunk_size=500, overlap=10)
    # for i, c in enumerate(fsep_tok.chunk_by_separator(document, separator="\\n\\n", encoding_name="cl100k_base")):
    #     print(f"[{i}] len≈{len(tiktoken.get_encoding('cl100k_base').encode(c))} toks ->", repr(c))

    # print("\n== Separator packing by characters (\\n\\n) ==")
    # fsep_char = FixedSizeChunking(chunk_size=800, overlap=10)
    # for i, c in enumerate(fsep_char.chunk_by_separator(document, separator="\\n\\n", encoding_name=None)):
    #     print(f"[{i}] ({len(c)} chars) ->", repr(c))
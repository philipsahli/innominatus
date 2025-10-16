# Night Thoughts - October 9, 2025

## Action Items & Open Points

### Documentation Cleanup
- [ ] **Verify all open points** from existing MD files (AZURE_OPENAI_DEMO_PROMPT.md, AZURE_OPENAI_IMPLEMENTATION_IMPROVED.md, etc.)
- [ ] **Move example files to examples/ folder**
  - Consider creating script to automate this process
  - Examples moved by script should go into appropriate subfolders
  - Maintain logical organization structure

### Graph System & Visualization

#### Core Concept
- **Early graph building is crucial** - need to identify logic patterns early in the process
- The graph system idea is fundamental to understanding workflow relationships

#### Current Issues
- [ ] **Workflow visualization broken** - current workflow does not show the links between steps
- [ ] **Improve workflow graph visualizations**
  - Links/edges not rendering properly
  - Need better representation of step dependencies
  - Consider interactive graph rendering

### RAG (Retrieval-Augmented Generation) Implementation

#### Database Integration
- [ ] **Implement RAG in database** instead of in-memory
  - Persistent storage for document embeddings
  - Better scalability and performance
  - Support for document versioning

#### Leader Election for Document Loading
- [ ] **Implement leader election mechanism**
  - Only the leader pod/instance loads documents to RAG
  - Prevents duplicate work and race conditions
  - Critical for multi-replica deployments

#### Incremental Document Loading
- [ ] **Load documents only when changed**
  - Track document modification timestamps
  - Compare against last indexed version
  - Avoid unnecessary re-indexing
  - Enables incremental code additions without full re-index

### Benefits of This Approach
- **Scalability**: Database-backed RAG allows horizontal scaling
- **Efficiency**: Only changed documents are processed
- **Reliability**: Leader election prevents conflicts
- **Performance**: Incremental updates instead of full reloads

## Technical Considerations

### RAG Database Schema (Proposed)
```sql
-- Document metadata and content tracking
CREATE TABLE rag_documents (
    id SERIAL PRIMARY KEY,
    file_path TEXT UNIQUE NOT NULL,
    content_hash TEXT NOT NULL,
    last_modified TIMESTAMP NOT NULL,
    last_indexed TIMESTAMP,
    embedding_model TEXT,
    INDEX idx_path (file_path),
    INDEX idx_modified (last_modified)
);

-- Vector embeddings storage
CREATE TABLE rag_embeddings (
    id SERIAL PRIMARY KEY,
    document_id INTEGER REFERENCES rag_documents(id),
    chunk_index INTEGER,
    embedding VECTOR(1536),  -- Dimension depends on model
    content_text TEXT,
    metadata JSONB
);
```

### Leader Election Implementation
- Use Kubernetes leader election library (`client-go/tools/leaderelection`)
- Store lease in ConfigMap or Lease resource
- Leader performs RAG indexing tasks
- Followers wait until leadership acquired

### Document Change Detection
```go
// Pseudocode for incremental loading
func LoadChangedDocuments(ctx context.Context) {
    files := DiscoverSourceFiles()
    for _, file := range files {
        currentHash := ComputeHash(file)
        lastIndexed := GetLastIndexedHash(file.Path)

        if currentHash != lastIndexed {
            // Document changed, re-index
            IndexDocument(ctx, file)
            UpdateDocumentRecord(file.Path, currentHash)
        }
    }
}
```

## Next Steps Priority

1. **High Priority**
   - Fix workflow graph visualization (blocking understanding)
   - Verify and consolidate open points from existing MD files

2. **Medium Priority**
   - Implement RAG database backend
   - Add leader election for RAG indexing

3. **Low Priority**
   - Organize examples folder structure
   - Create automation scripts for file organization

## Related Files
- AZURE_OPENAI_DEMO_PROMPT.md
- AZURE_OPENAI_IMPLEMENTATION_IMPROVED.md
- docs/GOLDEN_PATHS_METADATA.md
- workflows/*.yaml

## Azure OpenAI Integration Documentation

### Implementation Guides Created
Comprehensive documentation for integrating Azure OpenAI as an alternative embedding provider:

**Branch**: `feature/workflow-graph-viz`
**Commit**: [`f3f600d`](../../commit/f3f600d) - docs: add Azure OpenAI integration implementation guides

**Files Added**:
1. **AZURE_OPENAI_DEMO_PROMPT.md** - Initial implementation prompt
   - Azure OpenAI cloud service configuration
   - Demo environment integration (demo-time, demo-status, demo-nuke)
   - Kubernetes ConfigMap/Secret management
   - Optional component with fallback to standard OpenAI

2. **AZURE_OPENAI_IMPLEMENTATION_IMPROVED.md** - KISS & SOLID implementation guide
   - Provider abstraction pattern (`embedding_config.go`)
   - Dependency injection for testability
   - Single source of truth for embedding configuration
   - Extensible design for adding new providers (Google Vertex AI, etc.)
   - Component-based demo integration

**Key Design Principles**:
- **KISS**: Single configuration source, no duplication, clear separation
- **SOLID**: SRP (single responsibility), OCP (open/closed), DIP (dependency inversion)
- **Testability**: Easy to inject mock providers for testing
- **Extensibility**: Add new providers without modifying existing code

**Next Steps**:
- [ ] Implement `internal/ai/embedding_config.go` with provider abstraction
- [ ] Update `internal/ai/service.go` to use dependency injection
- [ ] Create `internal/cli/demo_components.go` with component interface
- [ ] Update `innominatus-ai-sdk` to support `CustomHeaders` in RAG config
- [ ] Add unit tests for `NewEmbeddingProvider()`
- [ ] Add integration tests with real Azure OpenAI resource

---

*Captured: 2025-10-09*
*Updated: 2025-10-09 18:32* - Added Azure OpenAI integration documentation reference
*Status: Draft - Action items to be prioritized and scheduled*

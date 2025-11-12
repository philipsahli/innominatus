declare module 'react-cytoscapejs' {
  import { Component } from 'react';
  import Cytoscape from 'cytoscape';

  export interface CytoscapeComponentProps {
    elements?: any[];
    style?: React.CSSProperties;
    layout?: any;
    stylesheet?: any;
    cy?: (cy: Cytoscape.Core) => void;
    pan?: { x: number; y: number };
    zoom?: number;
    minZoom?: number;
    maxZoom?: number;
    autoungrabify?: boolean;
    autounselectify?: boolean;
    boxSelectionEnabled?: boolean;
  }

  export default class CytoscapeComponent extends Component<CytoscapeComponentProps> {}
}

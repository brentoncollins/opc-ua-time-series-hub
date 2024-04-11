// src/App.js
import React from 'react';
import './App.css';
import NodeTree from './NodeTree';

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <h1>Nodes Hierarchy</h1>
        <NodeTree />
      </header>
    </div>
  );
}

export default App;

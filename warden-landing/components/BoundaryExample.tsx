const withoutWarden = [
  ["Read ./src", "Readable"],
  ["Read .env", "Readable"],
  ["Connect to api.github.com", "Reachable"],
  ["Connect to api.openai.com", "Reachable"],
];

const withWarden = [
  ["Read ./src", "Allowed"],
  ["Read .env", "Denied"],
  ["Connect to api.github.com", "Allowed"],
  ["Connect to api.openai.com", "Denied"],
];

export default function BoundaryExample() {
  return (
    <section id="boundary" className="home-section">
      <h2 className="accent-heading">Same tool. Smaller boundary.</h2>
      <p className="home-intro">
        In this example, the policy grants access to source files and one API.
        The sandbox blocks the other requests.
      </p>
      <div className="home-grid home-comparison">
        <Comparison caption="Without Warden" rows={withoutWarden} />
        <Comparison caption="With Warden" rows={withWarden} />
      </div>
      <a className="home-section-link" href="/videos/warden-zero-config.mp4">
        Watch the Filesystem MCP demo <span aria-hidden="true">→</span>
      </a>
    </section>
  );
}

function Comparison({ caption, rows }: { caption: string; rows: string[][] }) {
  return (
    <table>
      <caption>{caption}</caption>
      <thead className="sr-only">
        <tr><th scope="col">Request</th><th scope="col">Result</th></tr>
      </thead>
      <tbody>
        {rows.map(([request, result]) => (
          <tr key={request}>
            <th scope="row">{request}</th>
            <td>{result}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

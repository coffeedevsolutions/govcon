export default async function DbTestPage() {
  const res = await fetch("http://localhost:3000/api/db-test", { cache: "no-store" });
  const data = await res.json();

  return (
    <main style={{ padding: 24 }}>
      <h1>DB → Server → Client Test</h1>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </main>
  );
}


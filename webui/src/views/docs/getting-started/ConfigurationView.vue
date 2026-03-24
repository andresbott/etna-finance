<script setup lang="ts">
import Card from 'primevue/card'
import Message from 'primevue/message'
</script>

<template>
    <div class="doc-page">
        <Card>
            <template #title>Configuration</template>
            <template #content>
                <div class="doc-content">
                    <section>
                        <h3>Overview</h3>
                        <p>
                            Etna Finance is configured through a <code>config.yaml</code> file. The file is optional —
                            the application runs with sensible defaults if no configuration is provided.
                        </p>
                        <p>
                            Configuration is loaded in the following order, where later sources override earlier ones:
                        </p>
                        <ol>
                            <li>Built-in defaults</li>
                            <li><code>.env</code> file (optional)</li>
                            <li><code>config.yaml</code> file (optional)</li>
                            <li>Environment variables (prefix: <code>ETNA_</code>)</li>
                        </ol>
                    </section>

                    <section>
                        <h3>Generating a config file</h3>
                        <p>
                            The easiest way to create a configuration file is with the CLI:
                        </p>
                        <pre><code>etna config</code></pre>
                        <p>
                            This generates a fully commented <code>config.yaml</code> with all defaults in the current
                            directory. You can specify an output path with the <code>-o</code> flag:
                        </p>
                        <pre><code>etna config -o /path/to/config.yaml</code></pre>
                        <Message severity="info" :closable="false" icon="ti ti-info-circle">
                            The command will not overwrite an existing file.
                        </Message>
                    </section>

                    <section>
                        <h3>Starting the server</h3>
                        <p>
                            By default, the server looks for <code>config.yaml</code> in the current directory:
                        </p>
                        <pre><code>etna start</code></pre>
                        <p>
                            To use a config file at a different location:
                        </p>
                        <pre><code>etna start -c /path/to/config.yaml</code></pre>
                    </section>

                    <section>
                        <h3>Configuration reference</h3>

                        <h4>Server</h4>
                        <p>
                            Controls the main HTTP server that serves the application.
                        </p>
                        <ul>
                            <li><code>Port</code> — Port to listen on. Default: <code>8085</code>.</li>
                            <li><code>BindIp</code> — IP address to bind to. Default: empty (listen on all interfaces).</li>
                        </ul>

                        <h4>DataDir</h4>
                        <p>
                            Directory where the database, backups, attachments, and task logs are stored.
                            Default: <code>./data</code>. Created automatically if it does not exist.
                        </p>

                        <h4>Settings</h4>
                        <ul>
                            <li><code>MainCurrency</code> — Your primary currency as an ISO 4217 code (e.g. <code>EUR</code>, <code>USD</code>). Default: <code>CHF</code>.</li>
                            <li><code>AdditionalCurrencies</code> — A list of extra currencies to track alongside the main one.</li>
                            <li><code>DateFormat</code> — Display format for dates using <code>YYYY</code>, <code>MM</code>, <code>DD</code> tokens separated by <code>-</code>, <code>/</code>, or <code>.</code>. Default: <code>YYYY-MM-DD</code>.</li>
                            <li><code>InvestmentInstruments</code> — Enable investment tracking (stocks, ETFs, restricted stock assets). Default: <code>false</code>.</li>
                            <li><code>FinancialSimulator</code> — Enable financial simulator (portfolio simulator, real-estate simulator, etc.). Default: <code>false</code>.</li>
                            <li><code>MaxAttachmentSizeMB</code> — Maximum upload size for attachments in MB. Default: <code>10</code>.</li>
                        </ul>

                        <h4>Auth</h4>
                        <p>
                            Authentication is disabled by default — the application runs as a single user without
                            requiring login. To enable it:
                        </p>
                        <ul>
                            <li><code>Enabled</code> — Set to <code>true</code> to require login. Default: <code>false</code>.</li>
                            <li><code>DefaultUser</code> — Username used when auth is disabled. Default: <code>default</code>.</li>
                            <li><code>HashKey</code> — 64-character key for session encryption. Auto-generated if empty.</li>
                            <li><code>BlockKey</code> — 32-character key for session encryption. Auto-generated if empty.</li>
                            <li><code>UserStore</code> — Defines where users are stored:
                                <ul>
                                    <li><code>Type: static</code> — Users are defined directly in the config file.</li>
                                    <li><code>Type: file</code> — Users are loaded from a separate file specified by <code>Path</code>.</li>
                                </ul>
                            </li>
                        </ul>
                        <Message severity="warn" :closable="false" icon="ti ti-alert-triangle">
                            If <code>HashKey</code> and <code>BlockKey</code> are left empty, random keys are generated
                            at startup. This means user sessions will not survive application restarts.
                        </Message>

                        <h4>Env</h4>
                        <ul>
                            <li><code>LogLevel</code> — Logging verbosity: <code>debug</code>, <code>info</code>, <code>warn</code>, or <code>error</code>. Default: <code>info</code>.</li>
                            <li><code>Production</code> — Set to <code>true</code> for production deployments. Default: <code>false</code>.</li>
                        </ul>

                        <h4>Observability</h4>
                        <p>
                            A separate HTTP server for health checks and metrics.
                        </p>
                        <ul>
                            <li><code>Port</code> — Default: <code>9090</code>.</li>
                            <li><code>BindIp</code> — Default: empty (listen on all interfaces).</li>
                        </ul>

                        <h4>MarketDataImporters</h4>
                        <p>
                            Configuration for external market data providers used to fetch stock prices and
                            currency exchange rates.
                        </p>
                        <ul>
                            <li><code>Massive.ApiKeys</code> — A list of API keys for the market data provider. Multiple keys can be provided for rotation.</li>
                        </ul>
                    </section>

                    <section>
                        <h3>Environment variables</h3>
                        <p>
                            Any configuration value can be overridden with an environment variable using the
                            <code>ETNA_</code> prefix. Nested fields use underscores as separators. For example:
                        </p>
                        <pre><code>ETNA_SERVER_PORT=8085
ETNA_SETTINGS_MAINCURRENCY=USD
ETNA_SETTINGS_INSTRUMENTS=true
ETNA_AUTH_ENABLED=true
ETNA_ENV_LOGLEVEL=debug</code></pre>
                    </section>
                </div>
            </template>
        </Card>
    </div>
</template>

<style src="../doc-content.css"></style>

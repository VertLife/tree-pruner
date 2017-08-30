#!/usr/bin/perl

use strict;
use DBI;
use JSON;

my $database = "birdtree";
my $username = "";
my $password = "";

my %frmVals = ();
my $ret = &ReadForm(\%frmVals);
my $dbh = &ConnectToPg($database, $username, $password);

my $callback = $frmVals{'callback'};

if ($callback) { 
    print 'Access-Control-Allow-Origin: *';
    print 'Access-Control-Allow-Methods: GET'; 
    print "Content-type: application/javascript\n\n";
} else { 
    # Header for access via browser, curl, etc. 
    print "Content-type: application/json\n\n"; 
}

&listTreesets($dbh, $callback);

$dbh->disconnect;
exit;


# return JSON for the available datasets
#==============================================================
sub listTreesets {
	my ($dbh, $callback) = @_;

	my $statement = 'SELECT ts.treeset_id, ts.treeset_name, ';
	$statement .= '(SELECT count(*) FROM tree tr WHERE tr.treeset_id = ts.treeset_id) AS "Number of Trees", ';
	$statement .= '(SELECT count(*) FROM taxon_treeset tt WHERE tt.treeset_id = ts.treeset_id ) AS "Number of Taxa" ';
	$statement .= 'FROM treeset ts ORDER BY ts.treeset_name ';
	my $sth = $dbh->prepare( $statement ) or die "Can't prepare $statement: $dbh->errstr\n";		
	my $rv = $sth->execute or die "can't execute the query: $sth->errstr\n";

	my %json;
	while(my @row = $sth->fetchrow_array) {
		push @{ $json{treesets} }, { "name" => $row[1], "id" => $row[0], "tree_count" => $row[2], "OTU_each" => $row[3] } ;
	}
	my $rd = $sth->finish;
	my $json = to_json(\%json, {pretty=>'1'}); 
	if ($callback) { 
    	print $callback . '(' . $json . ');'; 
	} else { 
		print $json,"\n"; 
	}
}

#==============================================================
sub ReadForm {
	my ($varRef) = @_;
	my ($i, $key, $val, $variable, @list);
	
	if (&MethGet) {
		$variable = $ENV{'QUERY_STRING'};
	} elsif (&MethPost) {
		read(STDIN, $variable, $ENV{'CONTENT_LENGTH'});
	}
	
	@list = split(/[&;]/,$variable);
	
	foreach $i (0 .. $#list) {
		$list[$i] =~ s/\+/ /g;
		($key, $val) = split(/=/,$list[$i],2);
		$key =~ s/%(..)/pack("c",hex($1))/ge;
		$val =~ s/%(..)/pack("c",hex($1))/ge;
		$$varRef{$key} .= "\0" if (defined($$varRef{$key}));
		$$varRef{$key} .= $val;
	}
	return scalar($#list + 1);
}
sub MethGet { return ($ENV{'REQUEST_METHOD'} eq "GET") };
sub MethPost { return ($ENV{'REQUEST_METHOD'} eq "POST") };

# Connect to Postgres using DBI
#==============================================================
sub ConnectToPg {
 
    my ($cstr, $user, $pass) = @_;
  
    $cstr = "DBI:Pg:dbname="."$cstr";
    $cstr .= ";host=litoria.eeb.yale.edu";
  
    my $dbh = DBI->connect($cstr, $user, $pass, {PrintError => 1, RaiseError => 1});
    $dbh || &error("DBI connect failed : ",$dbh->errstr);
 
    return($dbh);
}


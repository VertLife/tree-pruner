#!/usr/bin/perl

use strict;
use DBI;

my $database = "birdtree";
my $username = "";
my $password = "";

my %frmVals = ();
my $ret = &ReadForm(\%frmVals);
my $dbh = &ConnectToPg($database, $username, $password);

print "Content-type: application/json\n\n";

# &listTreesets($dbh) if ( $frmVals{'treesets'} eq 'list' );
&listTreesets($dbh);

$dbh->disconnect;
exit;


# provide a popup list of the available datasets
#==============================================================
sub listTreesets {
	my ($dbh) = @_;

	my $statement = 'SELECT ts.treeset_id, ts.treeset_name, ';
	$statement .= '(SELECT count(*) FROM tree tr WHERE tr.treeset_id = ts.treeset_id) AS "Number of Trees", ';
	$statement .= '(SELECT count(*) FROM taxon_treeset tt WHERE tt.treeset_id = ts.treeset_id ) AS "Number of Taxa" ';
	$statement .= 'FROM treeset ts ORDER BY ts.treeset_name ';
	my $sth = $dbh->prepare( $statement ) or die "Can't prepare $statement: $dbh->errstr\n";		
	my $rv = $sth->execute or die "can't execute the query: $sth->errstr\n";

	my @values;
	my %labels;

	my @json;
	print q|{"treesets" : [|;
	while(my @row = $sth->fetchrow_array) {
		push @json, sprintf (qq|{"name": "%s", "id": %d, "tree_count": %d ,  "OTU_each" : %d }|, $row[1], $row[0], $row[2], $row[3]);
	}
	my $rd = $sth->finish;
	print join (', ',@json);
	print q|]}|;
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


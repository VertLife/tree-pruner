#!/usr/bin/python2.7
# -*- coding: utf-8 -*-

import os
import logging
import random
import json
import uuid
import webapp2

import cloudstorage as gcs

from google.appengine.api import app_identity
from google.appengine.ext import deferred
from google.appengine.ext import ndb

from ete2 import Tree


"""class PruneJob(ndb.Model):
   
    job_id = ndb.StringProperty()
    created_at = ndb.DateTimeProperty(auto_now_add=True)
    updated_at = ndb.DateTimeProperty(auto_now_add=True)
    total_trees ndb.NumberProperty()
    trees_completed = ndb.NumberProperty()
    status = ndb.StringProperty()

    @classmethod
    def tree_completed(cls, tree_id):
        cls.trees_completed = cls.trees_completed + 1
    
class PruneJob(ndb.Model):
  
    job_id = ndb.StringProperty()
    created_at = ndb.DateTimeProperty(auto_now_add=True)
    updated_at = ndb.DateTimeProperty(auto_now_add=True)
    total_trees ndb.NumberProperty()
    trees_completed = ndb.NumberProperty()
    status = ndb.StringProperty()

    @classmethod
    def tree_completed(cls, tree_id):
        cls.trees_completed = cls.trees_completed + 1

    @classmethod
    def tree_completed(cls, tree_id):
        cls.trees_completed = cls.trees_completed + 1  

"""
   

def create_file(filename, contents):
    """Create a file.

    The retry_params specified in the open call will override the default
    retry params for this particular file handle.

    Args:
    filename: filename.
    """
    write_retry_params = gcs.RetryParams(backoff_factor=1.1)
    gcs_file = gcs.open(filename,
        'w',
        content_type='text/plain',
        options={'x-goog-meta-foo': 'foo',
                'x-goog-meta-bar': 'bar'},
        retry_params=write_retry_params)
    gcs_file.write(contents)
    gcs_file.close()


def processTree(tree_filename, job_id, names):
    try:
        gcs_file = gcs.open('/data.vertlife.org/processing/birdtree/sources/single_trees/' + tree_filename)
        tree_file = gcs_file.read()
        gcs_file.close()
        tree = Tree(tree_file)
        missing = None
        logging.info(names)
        try:
            tree.prune(names, preserve_branch_length=True)
        except Exception as a:
            missing_names = str(a).replace('Node names not found: ','')
            names = filter(lambda name: name not in missing_names, names)
            tree.prune(names)

        filename = '/data.vertlife.org/pruned_treesets/' + job_id + '/' + tree_filename
        create_file(filename, tree.write())
    except Exception as e:
        logging.error(e)


class PruningJobHandler(webapp2.RequestHandler):
    def get(self):
        self.process_request()

    def post(self):
        self.process_request()

    def process_request(self):
        def generateTreeFile(prefix):
            treefile = '%s_%s.tre' % (prefix, "{:0>4d}".format(
                random.randrange(0, 9999, 1)))
            return treefile

        sample_size = int(self.request.get('sample_size', 10))

        tree_set = str(self.request.get('tree_set', 'Stage2_Parrot'))

        names = map(
            lambda n: n.replace(' ', '_'),
            str(self.request.get('species', '')).split(', ')
        )

        tree_files = {}
        pruned_trees = []
        job_id = 'tree-pruner-%s' % uuid.uuid4()
        counter = 0
        while len(tree_files.keys()) < sample_size and counter < 10000:

            tree_file = generateTreeFile(tree_set)
            while tree_files.has_key(tree_file):
                tree_file = generateTreeFile(tree_set)
            tree_files[tree_file] = True
            counter = counter + 1
            logging.info("Spawned task %s, %s" % (counter, tree_file))
            deferred.defer(processTree, tree_file, job_id, names)

        self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        self.response.headers['Content-Type'] = 'application/json'
        self.response.out.write(json.dumps({'job_id': job_id}))


class PruningResultHandler(webapp2.RequestHandler):
    def get(self):
        folder = '/data.vertlife.org/pruned_treesets'
        job_id = self.request.get('job_id')
        trees = gcs.listbucket(folder + '/' + job_id, max_keys=10000)
        for tree in trees:
            pruned_tree = gcs.open(tree)
            self.response.out.write(pruned_tree.read())
            pruned_tree.close()


application = webapp2.WSGIApplication(
    [('/api/prune', PruningJobHandler),
     ('/api/result', PruningResultHandler)
    ], debug=True)
